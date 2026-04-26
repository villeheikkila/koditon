package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	passkeysvc "koditon-go/internal/auth/passkey"
	db "koditon-go/internal/db"

	"github.com/go-webauthn/webauthn/protocol"
	wbauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFinishPasskeyAuthentication_SuccessCreatesSessionAndUpdatesPasskey(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	userID, credentialID, challengeID := createPasskeyAuthFixture(t, ctx, pool, queries)
	deviceID := uuid.New()

	resp, err := service.FinishPasskeyAuthentication(ctx, FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
		DeviceID:    deviceID,
		DeviceName:  "Test iPhone",
		DeviceOS:    "iOS 26",
		IP:          "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("finish passkey auth: %v", err)
	}
	if resp.UserID != userID {
		t.Fatalf("expected user %s, got %s", userID, resp.UserID)
	}
	if resp.SessionID == uuid.Nil {
		t.Fatal("expected a session id")
	}
	session, err := queries.GetSessionByID(ctx, resp.SessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.DeviceSessionUserAgent != nil {
		t.Fatalf("expected empty user agent, got %q", *session.DeviceSessionUserAgent)
	}

	passkeys, err := queries.ListPasskeysByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(passkeys) != 1 {
		t.Fatalf("expected 1 passkey, got %d", len(passkeys))
	}
	if passkeys[0].UserPasskeyCredentialIDB64url != credentialID {
		t.Fatalf("expected credential %s, got %s", credentialID, passkeys[0].UserPasskeyCredentialIDB64url)
	}
	if passkeys[0].UserPasskeySignCount != 42 {
		t.Fatalf("expected sign count 42, got %d", passkeys[0].UserPasskeySignCount)
	}
	if passkeys[0].UserPasskeyBackupState == nil || !*passkeys[0].UserPasskeyBackupState {
		t.Fatal("expected backup state to be updated")
	}
	if passkeys[0].UserPasskeyLastUsedAt == nil {
		t.Fatal("expected last used timestamp to be updated")
	}
}

func TestFinishPasskeyAuthentication_SanitizesSessionUserAgent(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	_, _, challengeID := createPasskeyAuthFixture(t, ctx, pool, queries)
	resp, err := service.FinishPasskeyAuthentication(ctx, FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
		DeviceID:    uuid.New(),
		UserAgent:   "  Koditon/1.0\t(\niPhone; iOS 26.0)\u0000  ",
	})
	if err != nil {
		t.Fatalf("finish passkey auth: %v", err)
	}

	session, err := queries.GetSessionByID(ctx, resp.SessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.DeviceSessionUserAgent == nil {
		t.Fatal("expected sanitized user agent")
	}
	want := "Koditon/1.0 ( iPhone; iOS 26.0)"
	if *session.DeviceSessionUserAgent != want {
		t.Fatalf("expected %q, got %q", want, *session.DeviceSessionUserAgent)
	}
}

func TestFinishPasskeyAuthentication_ReplayedChallengeFails(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	_, _, challengeID := createPasskeyAuthFixture(t, ctx, pool, queries)
	req := FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
		DeviceID:    uuid.New(),
	}

	if _, err := service.FinishPasskeyAuthentication(ctx, req); err != nil {
		t.Fatalf("first passkey auth failed: %v", err)
	}

	if _, err := service.FinishPasskeyAuthentication(ctx, req); !errors.Is(err, ErrPasskeyChallenge) {
		t.Fatalf("expected ErrPasskeyChallenge, got %v", err)
	}
}

func TestFinishPasskeyAuthentication_UnknownCredentialReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:  slog.Default(),
		pool:    pool,
		queries: queries,
		passkeyService: stubPasskeyCeremonyService{
			assertionResult: &wbauthn.Credential{
				Authenticator: wbauthn.Authenticator{SignCount: 7},
				Flags:         wbauthn.NewCredentialFlags(protocol.FlagUserPresent | protocol.FlagUserVerified),
			},
			skipLookup: true,
		},
	}

	challengeID := createPasskeyChallenge(t, ctx, queries, passkeyFlowAuthenticate, uuid.Nil, time.Now().Add(time.Minute))
	_, err := service.FinishPasskeyAuthentication(ctx, FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
		DeviceID:    uuid.New(),
	})
	if !errors.Is(err, ErrPasskeyNotFound) {
		t.Fatalf("expected ErrPasskeyNotFound, got %v", err)
	}
}

func TestFinishPasskeyAuthentication_ExpiredChallengeFails(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	challengeID := createPasskeyChallenge(t, ctx, queries, passkeyFlowAuthenticate, uuid.Nil, time.Now().Add(-time.Minute))
	_, err := service.FinishPasskeyAuthentication(ctx, FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
		DeviceID:    uuid.New(),
	})
	if !errors.Is(err, ErrPasskeyChallenge) {
		t.Fatalf("expected ErrPasskeyChallenge, got %v", err)
	}
}

func TestFinishPasskeyRegistration_RejectsMismatchedUser(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	owner := createAuthTestUser(t, ctx, pool, queries)
	other := createAuthTestUser(t, ctx, pool, queries)
	challengeID := createPasskeyChallenge(t, ctx, queries, passkeyFlowRegister, owner, time.Now().Add(time.Minute))

	_, err := service.FinishPasskeyRegistration(ctx, FinishPasskeyRegistrationRequest{
		UserID:      other,
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestFinishPasskeyRegistration_CreatesIdentityAndPasskey(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	userID := createAuthTestUser(t, ctx, pool, queries)
	challengeID := createPasskeyChallenge(t, ctx, queries, passkeyFlowRegister, userID, time.Now().Add(time.Minute))

	resp, err := service.FinishPasskeyRegistration(ctx, FinishPasskeyRegistrationRequest{
		UserID:      userID,
		ChallengeID: challengeID,
		Credential:  json.RawMessage(`{"type":"public-key"}`),
	})
	if err != nil {
		t.Fatalf("finish registration: %v", err)
	}
	if resp.CredentialID == "" {
		t.Fatal("expected credential id")
	}

	passkeys, err := queries.ListPasskeysByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(passkeys) != 1 {
		t.Fatalf("expected 1 passkey, got %d", len(passkeys))
	}
	if passkeys[0].UserPasskeyCredentialIDB64url != resp.CredentialID {
		t.Fatalf("expected credential id %s, got %s", resp.CredentialID, passkeys[0].UserPasskeyCredentialIDB64url)
	}
	if passkeys[0].UserPasskeyBackupEligible == nil || !*passkeys[0].UserPasskeyBackupEligible {
		t.Fatal("expected backup eligible flag")
	}
	if passkeys[0].UserPasskeyBackupState == nil || !*passkeys[0].UserPasskeyBackupState {
		t.Fatal("expected backup state flag")
	}
}

func TestBeginEmailAuthentication_ReturnsGenericResponseForExistingPasskeyAccount(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	userID := createAuthTestUser(t, ctx, pool, queries)
	email := "email-passkey@example.com"
	emailProvider := string(AuthProviderEmail)
	emailIdentity, err := queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserUuid:                  userID,
		UserIdentityProvider:      emailProvider,
		UserIdentityExternalID:    email,
		UserIdentityEmail:         &email,
		UserIdentityEmailVerified: true,
		UserIdentityData:          []byte(`{"source":"email"}`),
	})
	if err != nil {
		t.Fatalf("create email identity: %v", err)
	}
	passkeyProvider := string(AuthProviderPasskey)
	externalID := uuid.NewString()
	passkeyIdentity, err := queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserUuid:               userID,
		UserIdentityProvider:   passkeyProvider,
		UserIdentityExternalID: externalID,
		UserIdentityData:       []byte(`{"source":"test"}`),
	})
	if err != nil {
		t.Fatalf("create passkey identity: %v", err)
	}
	if emailIdentity.UserIdentityUuid == uuid.Nil {
		t.Fatal("expected email identity uuid")
	}

	rawCredentialID := []byte("credential-a")
	credentialID := base64.RawURLEncoding.EncodeToString(rawCredentialID)
	attestationType := "none"
	if _, err := queries.CreatePasskey(ctx, db.CreatePasskeyParams{
		UserUuid:                      userID,
		UserIdentityUuid:              passkeyIdentity.UserIdentityUuid,
		UserPasskeyCredentialID:       rawCredentialID,
		UserPasskeyCredentialIDB64url: credentialID,
		UserPasskeyPublicKey:          []byte("public-key"),
		UserPasskeyAttestationType:    attestationType,
		UserPasskeyTransports:         []string{"internal"},
		UserPasskeyUserHandle:         []byte("user-handle-a"),
		UserPasskeySignCount:          1,
		UserPasskeyLastUsedAt:         nil,
	}); err != nil {
		t.Fatalf("create passkey: %v", err)
	}

	resp, err := service.BeginEmailAuthentication(ctx, email)
	if err != nil {
		t.Fatalf("begin email auth: %v", err)
	}
	if resp.Method != EmailAuthenticationMethodMagicLink {
		t.Fatalf("expected generic magic_link method, got %s", resp.Method)
	}
	if resp.ChallengeID != uuid.Nil {
		t.Fatalf("expected no challenge id, got %s", resp.ChallengeID)
	}
	if len(resp.Options) > 0 {
		t.Fatalf("expected no passkey options, got %s", string(resp.Options))
	}
}

func TestBeginEmailAuthentication_ReturnsGenericResponseWithoutPasskeys(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := &Service{
		logger:         slog.Default(),
		pool:           pool,
		queries:        queries,
		passkeyService: stubPasskeyCeremonyService{},
	}

	userID := createAuthTestUser(t, ctx, pool, queries)
	email := "magic-link-only@example.com"
	if _, err := pool.Exec(ctx, "update users set user_email = $1 where user_uuid = $2", email, userID); err != nil {
		t.Fatalf("set user email: %v", err)
	}
	passkeyProvider := string(AuthProviderPasskey)
	externalID := uuid.NewString()
	passkeyIdentity, err := queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserUuid:               userID,
		UserIdentityProvider:   passkeyProvider,
		UserIdentityExternalID: externalID,
		UserIdentityData:       []byte(`{"source":"test"}`),
	})
	if err != nil {
		t.Fatalf("create passkey identity: %v", err)
	}
	rawCredentialID := []byte("credential-b")
	credentialID := base64.RawURLEncoding.EncodeToString(rawCredentialID)
	attestationType := "none"
	if _, err := queries.CreatePasskey(ctx, db.CreatePasskeyParams{
		UserUuid:                      userID,
		UserIdentityUuid:              passkeyIdentity.UserIdentityUuid,
		UserPasskeyCredentialID:       rawCredentialID,
		UserPasskeyCredentialIDB64url: credentialID,
		UserPasskeyPublicKey:          []byte("public-key"),
		UserPasskeyAttestationType:    attestationType,
		UserPasskeyTransports:         []string{"internal"},
		UserPasskeyUserHandle:         []byte("user-handle-b"),
		UserPasskeySignCount:          1,
		UserPasskeyLastUsedAt:         nil,
	}); err != nil {
		t.Fatalf("create passkey: %v", err)
	}

	resp, err := service.BeginEmailAuthentication(ctx, email)
	if err != nil {
		t.Fatalf("begin email auth: %v", err)
	}
	if resp.Method != EmailAuthenticationMethodMagicLink {
		t.Fatalf("expected magic_link method, got %s", resp.Method)
	}
	if resp.ChallengeID != uuid.Nil {
		t.Fatalf("expected no challenge id, got %s", resp.ChallengeID)
	}
	if len(resp.Options) > 0 {
		t.Fatalf("expected no passkey options, got %s", string(resp.Options))
	}
}

type stubPasskeyCeremonyService struct {
	assertionResult           *wbauthn.Credential
	skipLookup                bool
	beginAuthenticationCalled *bool
	beginAuthenticationName   *string
}

func (s stubPasskeyCeremonyService) BeginDiscoverableAuthentication() (*protocol.CredentialAssertion, *wbauthn.SessionData, error) {
	return &protocol.CredentialAssertion{}, &wbauthn.SessionData{Challenge: "stub-authentication"}, nil
}

func (s stubPasskeyCeremonyService) BeginAuthentication(user passkeysvc.User) (*protocol.CredentialAssertion, *wbauthn.SessionData, error) {
	if s.beginAuthenticationCalled != nil {
		*s.beginAuthenticationCalled = true
	}
	if s.beginAuthenticationName != nil {
		*s.beginAuthenticationName = user.Name
	}
	return &protocol.CredentialAssertion{}, &wbauthn.SessionData{Challenge: "stub-authentication"}, nil
}

func (s stubPasskeyCeremonyService) BeginRegistration(user passkeysvc.User, exclude []protocol.CredentialDescriptor) (*protocol.CredentialCreation, *wbauthn.SessionData, error) {
	_ = user
	_ = exclude
	return &protocol.CredentialCreation{}, &wbauthn.SessionData{Challenge: "stub-registration"}, nil
}

func (s stubPasskeyCeremonyService) FinishRegistration(ctx context.Context, user passkeysvc.User, session wbauthn.SessionData, credentialJSON []byte) (*wbauthn.Credential, error) {
	_ = ctx
	_ = user
	_ = session
	_ = credentialJSON
	flags := wbauthn.NewCredentialFlags(
		protocol.FlagUserPresent | protocol.FlagUserVerified | protocol.FlagBackupEligible | protocol.FlagBackupState,
	)
	aaguid := uuid.MustParse("fbfc3007-154e-4ecc-8c0b-6e020557d7bd")
	return &wbauthn.Credential{
		ID:              []byte("registered-passkey"),
		PublicKey:       []byte("public-key"),
		AttestationType: "none",
		Transport:       []protocol.AuthenticatorTransport{protocol.Internal},
		Flags:           flags,
		Authenticator: wbauthn.Authenticator{
			AAGUID:    aaguid[:],
			SignCount: 1,
		},
	}, nil
}

func (s stubPasskeyCeremonyService) FinishPasskeyLogin(
	ctx context.Context,
	session wbauthn.SessionData,
	credentialJSON []byte,
	handler wbauthn.DiscoverableUserHandler,
) (wbauthn.User, *wbauthn.Credential, error) {
	_ = ctx
	_ = session
	_ = credentialJSON

	result := s.assertionResult
	if result == nil {
		result = &wbauthn.Credential{
			Authenticator: wbauthn.Authenticator{SignCount: 42},
			Flags: wbauthn.NewCredentialFlags(
				protocol.FlagUserPresent | protocol.FlagUserVerified | protocol.FlagBackupEligible | protocol.FlagBackupState,
			),
		}
	}

	if s.skipLookup {
		return nil, result, nil
	}

	user, err := handler([]byte("credential-a"), []byte("user-handle-a"))
	return user, result, err
}

func openAuthTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL (or LOCAL_DATABASE_URL) to run DB-backed auth test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

func createAuthTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, queries *db.Queries) uuid.UUID {
	t.Helper()

	user, err := queries.CreateUser(ctx, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		if _, cleanupErr := pool.Exec(ctx, "delete from users where user_uuid = $1", user.UserUuid); cleanupErr != nil {
			t.Fatalf("cleanup user: %v", cleanupErr)
		}
	})
	return user.UserUuid
}

func createPasskeyAuthFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *db.Queries,
) (userID uuid.UUID, credentialID string, challengeID uuid.UUID) {
	t.Helper()

	userID = createAuthTestUser(t, ctx, pool, queries)
	provider := string(AuthProviderPasskey)
	externalID := uuid.NewString()
	identity, err := queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserUuid:               userID,
		UserIdentityProvider:   provider,
		UserIdentityExternalID: externalID,
		UserIdentityData:       []byte(`{"source":"test"}`),
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	rawCredentialID := []byte("credential-a")
	credentialID = base64.RawURLEncoding.EncodeToString(rawCredentialID)
	attestationType := "none"
	name := "Primary Passkey"
	backupEligible := true
	backupState := false
	flags := int32(
		wbauthn.NewCredentialFlags(protocol.FlagUserPresent | protocol.FlagUserVerified | protocol.FlagBackupEligible).ProtocolValue(),
	)
	if _, err := queries.CreatePasskey(ctx, db.CreatePasskeyParams{
		UserUuid:                      userID,
		UserIdentityUuid:              identity.UserIdentityUuid,
		UserPasskeyCredentialID:       rawCredentialID,
		UserPasskeyCredentialIDB64url: credentialID,
		UserPasskeyPublicKey:          []byte("public-key"),
		UserPasskeyAttestationType:    attestationType,
		UserPasskeyTransports:         []string{"internal"},
		UserPasskeyUserHandle:         []byte("user-handle-a"),
		UserPasskeySignCount:          1,
		UserPasskeyFlags:              &flags,
		UserPasskeyAaguid:             uuidPtr(uuid.MustParse("fbfc3007-154e-4ecc-8c0b-6e020557d7bd")),
		UserPasskeyName:               &name,
		UserPasskeyBackupEligible:     &backupEligible,
		UserPasskeyBackupState:        &backupState,
		UserPasskeyLastUsedAt:         nil,
	}); err != nil {
		t.Fatalf("create passkey: %v", err)
	}

	challengeID = createPasskeyChallenge(t, ctx, queries, passkeyFlowAuthenticate, userID, time.Now().Add(time.Minute))
	return userID, credentialID, challengeID
}

func createPasskeyChallenge(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
	flow string,
	userID uuid.UUID,
	expiresAt time.Time,
) uuid.UUID {
	t.Helper()

	sessionJSON, err := passkeysvc.MarshalSessionData(&wbauthn.SessionData{Challenge: "stub-session"})
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}

	params := db.CreateWebauthnChallengeParams{
		AuthWebauthnChallengeFlow:       flow,
		AuthWebauthnChallengeSession:    sessionJSON,
		AuthWebauthnChallengeExpiresAt:  expiresAt,
		AuthWebauthnChallengeUserHandle: []byte("user-handle-a"),
		AuthWebauthnChallengeDeviceID:   nil,
		UserUuid:                        nil,
	}
	if userID != uuid.Nil {
		params.UserUuid = &userID
		label := "Test User"
		params.AuthWebauthnChallengeUserDisplayName = &label
	}

	challenge, err := queries.CreateWebauthnChallenge(ctx, params)
	if err != nil {
		t.Fatalf("create webauthn challenge: %v", err)
	}
	return challenge.AuthWebauthnChallengeUuid
}

func uuidPtr(value uuid.UUID) *uuid.UUID {
	return &value
}
