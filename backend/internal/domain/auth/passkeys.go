package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	db "koditon/internal/db"
	"koditon/internal/domain/auth/passkey"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/useragent"

	"github.com/go-webauthn/webauthn/protocol"
	wbauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	passkeyFlowAuthenticate = "authenticate"
	passkeyFlowRegister     = "register"
	passkeyChallengeTTL     = 5 * time.Minute
)

type BeginPasskeyAuthenticateResponse struct {
	ChallengeID uuid.UUID
	Options     json.RawMessage
}

type FinishPasskeyAuthenticateRequest struct {
	ChallengeID      uuid.UUID
	Credential       json.RawMessage
	DeviceID         uuid.UUID
	DeviceName       string
	DeviceOS         string
	DeviceModel      string
	DeviceLocale     string
	DeviceTimeZone   string
	DeviceAppVersion string
	UserAgent        string
	IP               string
}

type FinishPasskeyAuthenticateResponse struct {
	UserID    uuid.UUID
	IsNewUser bool
	SessionID uuid.UUID
}

type BeginPasskeyRegistrationRequest struct {
	UserID    uuid.UUID
	DeviceID  uuid.UUID
	UserLabel string
}

type BeginPasskeyRegistrationResponse struct {
	ChallengeID uuid.UUID
	Options     json.RawMessage
}

type FinishPasskeyRegistrationRequest struct {
	UserID      uuid.UUID
	ChallengeID uuid.UUID
	Credential  json.RawMessage
}

type FinishPasskeyRegistrationResponse struct {
	CredentialID string
}

type ListPasskeyItem struct {
	CredentialID   string
	Name           string
	AAGUID         *uuid.UUID
	BackupEligible *bool
	BackupState    *bool
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

type PasskeyAccountUser struct {
	userID      uuid.UUID
	handle      []byte
	displayName string
	credentials []wbauthn.Credential
}

func (u PasskeyAccountUser) WebAuthnID() []byte {
	return u.handle
}

func (u PasskeyAccountUser) WebAuthnName() string {
	if u.displayName == "" {
		return u.userID.String()
	}
	return u.displayName
}

func (u PasskeyAccountUser) WebAuthnDisplayName() string {
	if u.displayName == "" {
		return u.userID.String()
	}
	return u.displayName
}

func (u PasskeyAccountUser) WebAuthnCredentials() []wbauthn.Credential {
	return u.credentials
}

func (s *Service) BeginPasskeyAuthentication(ctx context.Context) (*BeginPasskeyAuthenticateResponse, error) {
	if s.passkeyService == nil {
		return nil, ErrPasskeyConfig
	}
	assertion, session, err := s.passkeyService.BeginDiscoverableAuthentication()
	if err != nil {
		return nil, fmt.Errorf("begin discoverable login: %w", err)
	}
	return s.createPasskeyAuthenticationChallenge(ctx, assertion.Response, session, uuid.Nil, nil, nil)
}

func (s *Service) createPasskeyAuthenticationChallenge(
	ctx context.Context,
	assertion protocol.PublicKeyCredentialRequestOptions,
	session *wbauthn.SessionData,
	userID uuid.UUID,
	userHandle []byte,
	verifiedEmail *string,
) (*BeginPasskeyAuthenticateResponse, error) {
	sessionJSON, err := passkey.MarshalSessionData(session)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}
	displayName := (*string)(nil)
	var userUUID *uuid.UUID
	if userID != uuid.Nil {
		display := userID.String()
		displayName = &display
		userUUID = &userID
	}
	challenge, err := s.queries.CreateWebauthnChallenge(ctx, db.CreateWebauthnChallengeParams{
		AuthWebauthnChallengeFlow:            passkeyFlowAuthenticate,
		AuthWebauthnChallengeSession:         sessionJSON,
		AuthWebauthnChallengeExpiresAt:       time.Now().Add(passkeyChallengeTTL),
		AuthWebauthnChallengeUserHandle:      userHandle,
		AuthWebauthnChallengeUserDisplayName: displayName,
		AuthWebauthnChallengeVerifiedEmail:   verifiedEmail,
		AuthWebauthnChallengeDeviceID:        nil,
		UserUuid:                             userUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("create challenge: %w", err)
	}
	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return nil, fmt.Errorf("marshal assertion options: %w", err)
	}
	return &BeginPasskeyAuthenticateResponse{
		ChallengeID: challenge.AuthWebauthnChallengeUuid,
		Options:     optionsJSON,
	}, nil
}

func (s *Service) FinishPasskeyAuthentication(ctx context.Context, req FinishPasskeyAuthenticateRequest) (*FinishPasskeyAuthenticateResponse, error) {
	if s.passkeyService == nil {
		return nil, ErrPasskeyConfig
	}
	challenge, err := s.queries.ConsumeWebauthnChallenge(ctx, db.ConsumeWebauthnChallengeParams{
		AuthWebauthnChallengeUuid: req.ChallengeID,
		AuthWebauthnChallengeFlow: passkeyFlowAuthenticate,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasskeyChallenge
		}
		return nil, fmt.Errorf("consume challenge: %w", err)
	}
	sessionData, err := passkey.UnmarshalSessionData(challenge.AuthWebauthnChallengeSession)
	if err != nil {
		return nil, fmt.Errorf("unmarshal challenge session: %w", err)
	}
	var resolvedUserID uuid.UUID
	var resolvedPasskey db.GetPasskeyByUserHandleAndCredentialIDRow
	lookupHandler := func(rawID, userHandle []byte) (wbauthn.User, error) {
		row, getErr := s.queries.GetPasskeyByUserHandleAndCredentialID(ctx, db.GetPasskeyByUserHandleAndCredentialIDParams{
			UserPasskeyUserHandle:   userHandle,
			UserPasskeyCredentialID: rawID,
		})
		if getErr != nil {
			return nil, getErr
		}
		resolvedUserID = row.UserUuid
		resolvedPasskey = row
		return PasskeyAccountUser{
			userID:      row.UserUuid,
			handle:      row.UserPasskeyUserHandle,
			displayName: row.UserUuid.String(),
			credentials: []wbauthn.Credential{passkeyRowToCredential(row)},
		}, nil
	}

	_, parsedCredential, err := s.passkeyService.FinishPasskeyLogin(ctx, sessionData, req.Credential, lookupHandler)
	if err != nil {
		return nil, fmt.Errorf("finish passkey login: %w", err)
	}
	if resolvedUserID == uuid.Nil {
		return nil, ErrPasskeyNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
	qtx := s.queries.WithTx(tx)

	if err := qtx.UpdatePasskeyUsage(ctx, db.UpdatePasskeyUsageParams{
		UserPasskeySignCount:    int64(parsedCredential.Authenticator.SignCount),
		UserPasskeyCredentialID: resolvedPasskey.UserPasskeyCredentialID,
		UserPasskeyBackupState:  boolPtr(parsedCredential.Flags.BackupState),
	}); err != nil {
		return nil, fmt.Errorf("update passkey usage: %w", err)
	}

	sessionID, err := s.createSessionWithProvider(ctx, tx, createSessionParams{
		UserID:           resolvedUserID,
		Provider:         AuthProviderPasskey,
		DeviceID:         req.DeviceID,
		DeviceName:       req.DeviceName,
		DeviceOS:         req.DeviceOS,
		DeviceModel:      req.DeviceModel,
		DeviceLocale:     req.DeviceLocale,
		DeviceTimeZone:   req.DeviceTimeZone,
		DeviceAppVersion: req.DeviceAppVersion,
		UserAgent:        req.UserAgent,
		IP:               req.IP,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &FinishPasskeyAuthenticateResponse{UserID: resolvedUserID, SessionID: sessionID}, nil
}

func (s *Service) BeginPasskeyRegistration(ctx context.Context, req BeginPasskeyRegistrationRequest) (*BeginPasskeyRegistrationResponse, error) {
	if s.passkeyService == nil {
		return nil, ErrPasskeyConfig
	}
	passkeys, err := s.queries.ListPasskeysByUserID(ctx, req.UserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("list existing passkeys: %w", err)
	}
	handle := req.UserID[:]
	if len(passkeys) > 0 && len(passkeys[0].UserPasskeyUserHandle) > 0 {
		handle = passkeys[0].UserPasskeyUserHandle
	}
	existingCreds := make([]wbauthn.Credential, 0, len(passkeys))
	exclude := make([]protocol.CredentialDescriptor, 0, len(passkeys))
	for _, row := range passkeys {
		cred := passkeyListRowToCredential(row)
		existingCreds = append(existingCreds, cred)
		exclude = append(exclude, cred.Descriptor())
	}
	label := req.UserLabel
	if label == "" {
		label = req.UserID.String()
	}
	creation, session, err := s.passkeyService.BeginRegistration(passkey.User{
		ID:          handle,
		Name:        label,
		DisplayName: label,
		Credentials: existingCreds,
	}, exclude)
	if err != nil {
		return nil, fmt.Errorf("begin add-passkey registration: %w", err)
	}
	sessionJSON, err := passkey.MarshalSessionData(session)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}
	challenge, err := s.queries.CreateWebauthnChallenge(ctx, db.CreateWebauthnChallengeParams{
		AuthWebauthnChallengeFlow:            passkeyFlowRegister,
		AuthWebauthnChallengeSession:         sessionJSON,
		AuthWebauthnChallengeExpiresAt:       time.Now().Add(passkeyChallengeTTL),
		AuthWebauthnChallengeUserHandle:      handle,
		AuthWebauthnChallengeUserDisplayName: &label,
		AuthWebauthnChallengeDeviceID:        &req.DeviceID,
		UserUuid:                             &req.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("create challenge: %w", err)
	}
	optionsJSON, err := json.Marshal(creation.Response)
	if err != nil {
		return nil, fmt.Errorf("marshal creation options: %w", err)
	}
	return &BeginPasskeyRegistrationResponse{ChallengeID: challenge.AuthWebauthnChallengeUuid, Options: optionsJSON}, nil
}

func (s *Service) FinishPasskeyRegistration(ctx context.Context, req FinishPasskeyRegistrationRequest) (*FinishPasskeyRegistrationResponse, error) {
	logger := logging.With(s.logger, logging.Op("auth.passkey.finish_registration"), slog.Any("user_id", req.UserID), slog.Any("challenge_id", req.ChallengeID))
	if s.passkeyService == nil {
		return nil, ErrPasskeyConfig
	}
	challenge, err := s.queries.ConsumeWebauthnChallenge(ctx, db.ConsumeWebauthnChallengeParams{
		AuthWebauthnChallengeUuid: req.ChallengeID,
		AuthWebauthnChallengeFlow: passkeyFlowRegister,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasskeyChallenge
		}
		return nil, fmt.Errorf("consume challenge: %w", err)
	}
	if challenge.UserUuid != uuid.Nil && challenge.UserUuid != req.UserID {
		return nil, ErrUnauthorized
	}
	sessionData, err := passkey.UnmarshalSessionData(challenge.AuthWebauthnChallengeSession)
	if err != nil {
		return nil, fmt.Errorf("unmarshal challenge: %w", err)
	}
	passkeys, err := s.queries.ListPasskeysByUserID(ctx, req.UserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("load user passkeys: %w", err)
	}
	existing := make([]wbauthn.Credential, 0, len(passkeys))
	for _, row := range passkeys {
		existing = append(existing, passkeyListRowToCredential(row))
	}
	label := req.UserID.String()
	if challenge.AuthWebauthnChallengeUserDisplayName != nil {
		label = *challenge.AuthWebauthnChallengeUserDisplayName
	}
	credential, err := s.passkeyService.FinishRegistration(ctx, passkey.User{
		ID:          challenge.AuthWebauthnChallengeUserHandle,
		Name:        label,
		DisplayName: label,
		Credentials: existing,
	}, sessionData, req.Credential)
	if err != nil {
		logger.ErrorContext(ctx, "passkey finish registration failed", "error", err, "outcome", logging.OutcomeError)
		return nil, fmt.Errorf("finish registration: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
	qtx := s.queries.WithTx(tx)

	userEmail, err := qtx.GetUserEmailByUUID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user email: %w", err)
	}

	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	provider := AuthProviderPasskey
	identityData, _ := json.Marshal(map[string]any{
		"credential_id": credentialID,
		"aaguid":        uuidFromBytes(credential.Authenticator.AAGUID),
		"transports":    credentialTransports(credential.Transport),
		"backup": map[string]bool{
			"eligible": credential.Flags.BackupEligible,
			"state":    credential.Flags.BackupState,
		},
	})
	identity, err := qtx.CreateIdentity(ctx, db.CreateIdentityParams{
		UserUuid:                  req.UserID,
		UserIdentityProvider:      string(provider),
		UserIdentityExternalID:    credentialID,
		UserIdentityEmail:         userEmail,
		UserIdentityEmailVerified: userEmail != nil,
		UserIdentityData:          identityData,
	})
	if err != nil {
		return nil, fmt.Errorf("create passkey identity: %w", err)
	}

	if _, err := qtx.CreatePasskey(ctx, db.CreatePasskeyParams{
		UserUuid:                      req.UserID,
		UserIdentityUuid:              identity.UserIdentityUuid,
		UserPasskeyCredentialID:       credential.ID,
		UserPasskeyCredentialIDB64url: credentialID,
		UserPasskeyPublicKey:          credential.PublicKey,
		UserPasskeyAttestationType:    credential.AttestationType,
		UserPasskeyTransports:         credentialTransports(credential.Transport),
		UserPasskeyUserHandle:         challenge.AuthWebauthnChallengeUserHandle,
		UserPasskeySignCount:          int64(credential.Authenticator.SignCount),
		UserPasskeyFlags:              int32Ptr(int32(credential.Flags.ProtocolValue())),
		UserPasskeyAaguid:             uuidPtrFromBytes(credential.Authenticator.AAGUID),
		UserPasskeyName:               &label,
		UserPasskeyBackupEligible:     boolPtr(credential.Flags.BackupEligible),
		UserPasskeyBackupState:        boolPtr(credential.Flags.BackupState),
		UserPasskeyLastUsedAt:         nil,
	}); err != nil {
		return nil, fmt.Errorf("create passkey: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return &FinishPasskeyRegistrationResponse{CredentialID: credentialID}, nil
}

func (s *Service) ListPasskeys(ctx context.Context, userID uuid.UUID) ([]ListPasskeyItem, error) {
	rows, err := s.queries.ListPasskeysByUserID(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	items := make([]ListPasskeyItem, 0, len(rows))
	for _, row := range rows {
		name := "Passkey"
		if row.UserPasskeyName != nil && *row.UserPasskeyName != "" {
			name = *row.UserPasskeyName
		}
		var lastUsedAt *time.Time
		if row.UserPasskeyLastUsedAt != nil {
			lastUsedAt = row.UserPasskeyLastUsedAt
		}
		items = append(items, ListPasskeyItem{
			CredentialID:   row.UserPasskeyCredentialIDB64url,
			Name:           name,
			AAGUID:         uuidPtrFromBytes(bytesFromPgUUID(row.UserPasskeyAaguid)),
			BackupEligible: row.UserPasskeyBackupEligible,
			BackupState:    row.UserPasskeyBackupState,
			CreatedAt:      row.UserPasskeyCreatedAt,
			LastUsedAt:     lastUsedAt,
		})
	}
	return items, nil
}

func (s *Service) DeletePasskey(ctx context.Context, userID uuid.UUID, credentialID string) error {
	logger := logging.With(s.logger, logging.Op("auth.passkey.delete"), slog.Any("user_id", userID), slog.String("credential_id", credentialID))
	outcome, err := s.queries.RevokePasskeyByCredentialB64ForUser(ctx, db.RevokePasskeyByCredentialB64ForUserParams{
		UserUuid:                 userID,
		TargetCredentialIDB64url: credentialID,
	})
	if err != nil {
		return fmt.Errorf("revoke passkey: %w", err)
	}
	switch outcome {
	case "deleted":
		logger.InfoContext(ctx, "passkey deleted", "outcome", outcome)
		return nil
	case "last_passkey":
		logger.WarnContext(ctx, "passkey delete blocked", "outcome", outcome)
		return ErrPasskeyLast
	case "not_found":
		logger.WarnContext(ctx, "passkey delete failed", "outcome", outcome)
		return ErrPasskeyNotFound
	default:
		return fmt.Errorf("unexpected passkey delete outcome: %s", outcome)
	}
}

type createSessionParams struct {
	UserID           uuid.UUID
	Provider         AuthProvider
	DeviceID         uuid.UUID
	DeviceName       string
	DeviceOS         string
	DeviceModel      string
	DeviceLocale     string
	DeviceTimeZone   string
	DeviceAppVersion string
	UserAgent        string
	IP               string
}

func (s *Service) createSessionWithProvider(ctx context.Context, tx pgx.Tx, params createSessionParams) (uuid.UUID, error) {
	qtx := s.queries.WithTx(tx)
	var ip *string
	if params.IP != "" {
		if parsed, parseErr := netip.ParseAddr(params.IP); parseErr == nil {
			normalizedIP := parsed.String()
			ip = &normalizedIP
		}
	}
	var userAgent *string
	normalizedUserAgent := useragent.Normalize(params.UserAgent)
	if normalizedUserAgent != "" {
		userAgent = &normalizedUserAgent
	}
	deviceID := params.DeviceID
	if deviceID != uuid.Nil {
		var deviceName *string
		if params.DeviceName != "" {
			deviceName = &params.DeviceName
		}
		var deviceOS *string
		if params.DeviceOS != "" {
			deviceOS = &params.DeviceOS
		}
		var deviceAppVersion *string
		if params.DeviceAppVersion != "" {
			deviceAppVersion = &params.DeviceAppVersion
		}
		if _, err := qtx.UpsertDevice(ctx, db.UpsertDeviceParams{
			UserDeviceID:         deviceID,
			UserUuid:             params.UserID,
			UserDeviceName:       deviceName,
			UserDeviceOs:         deviceOS,
			UserDeviceAppVersion: deviceAppVersion,
		}); err != nil {
			return uuid.Nil, fmt.Errorf("upsert device: %w", err)
		}
		if err := s.updateDeviceMetadata(ctx, tx, deviceID, params); err != nil {
			return uuid.Nil, err
		}
	}
	provider := string(params.Provider)
	session, err := qtx.CreateSession(ctx, db.CreateSessionParams{
		UserUuid:                    params.UserID,
		DeviceSessionUserDeviceUuid: deviceID,
		DeviceSessionUserAgent:      userAgent,
		DeviceSessionIp:             ip,
		DeviceSessionProvider:       provider,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create session: %w", err)
	}
	if err := s.updateSessionMetadata(ctx, tx, session.DeviceSessionUuid, params); err != nil {
		return uuid.Nil, err
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventIssued,
		AuthType:  params.Provider,
		SessionID: session.DeviceSessionUuid,
		UserID:    params.UserID,
		TokenType: "session",
	})
	return session.DeviceSessionUuid, nil
}

func passkeyRowToCredential(row db.GetPasskeyByUserHandleAndCredentialIDRow) wbauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, 0, len(row.UserPasskeyTransports))
	for _, t := range row.UserPasskeyTransports {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	flags := protocol.AuthenticatorFlags(0)
	if row.UserPasskeyFlags != nil {
		flags = protocol.AuthenticatorFlags(*row.UserPasskeyFlags)
	}
	return wbauthn.Credential{
		ID:              row.UserPasskeyCredentialID,
		PublicKey:       row.UserPasskeyPublicKey,
		AttestationType: row.UserPasskeyAttestationType,
		Transport:       transports,
		Flags:           wbauthn.NewCredentialFlags(flags),
		Authenticator: wbauthn.Authenticator{
			AAGUID:    bytesFromPgUUID(row.UserPasskeyAaguid),
			SignCount: uint32(row.UserPasskeySignCount),
		},
	}
}

func passkeyListRowToCredential(row db.ListPasskeysByUserIDRow) wbauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, 0, len(row.UserPasskeyTransports))
	for _, t := range row.UserPasskeyTransports {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	flags := protocol.AuthenticatorFlags(0)
	if row.UserPasskeyFlags != nil {
		flags = protocol.AuthenticatorFlags(*row.UserPasskeyFlags)
	}
	return wbauthn.Credential{
		ID:              row.UserPasskeyCredentialID,
		PublicKey:       row.UserPasskeyPublicKey,
		AttestationType: row.UserPasskeyAttestationType,
		Transport:       transports,
		Flags:           wbauthn.NewCredentialFlags(flags),
		Authenticator: wbauthn.Authenticator{
			AAGUID:    bytesFromPgUUID(row.UserPasskeyAaguid),
			SignCount: uint32(row.UserPasskeySignCount),
		},
	}
}

func credentialTransports(values []protocol.AuthenticatorTransport) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, string(v))
	}
	return out
}

func uuidFromBytes(value []byte) *uuid.UUID {
	if len(value) != 16 {
		return nil
	}
	id, err := uuid.FromBytes(value)
	if err != nil {
		return nil
	}
	return &id
}

func uuidPtrFromBytes(value []byte) *uuid.UUID {
	if len(value) != 16 {
		return nil
	}
	id, err := uuid.FromBytes(value)
	if err != nil {
		return nil
	}
	return &id
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func int32Ptr(value int32) *int32 {
	v := value
	return &v
}

func bytesFromPgUUID(value *uuid.UUID) []byte {
	if value == nil {
		return nil
	}
	id := [16]byte(*value)
	return id[:]
}
