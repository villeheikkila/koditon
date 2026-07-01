package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	db "koditon/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	OAuthDeviceCodeTTL      = 10 * time.Minute
	OAuthDevicePollInterval = 5
)

type OAuthCreateDeviceAuthorizationRequest struct {
	ClientID string
	Scopes   []string
	Audience string
}

type OAuthCreateDeviceAuthorizationResponse struct {
	DeviceCode string
	UserCode   string
	ExpiresIn  int
	Interval   int
}

type OAuthApproveDeviceAuthorizationRequest struct {
	UserCode string
	UserID   uuid.UUID
}

type OAuthDenyDeviceAuthorizationRequest struct {
	UserCode string
	UserID   uuid.UUID
}

type OAuthExchangeDeviceCodeRequest struct {
	ClientID   string
	DeviceCode string
	Audience   string
}

type OAuthConsumeApprovedDeviceCodeRequest struct {
	ClientID   string
	DeviceCode string
}

type OAuthDeviceAuthorizationStatus string

const (
	OAuthDeviceAuthorizationStatusPending OAuthDeviceAuthorizationStatus = "pending"
	OAuthDeviceAuthorizationStatusDenied  OAuthDeviceAuthorizationStatus = "denied"
	OAuthDeviceAuthorizationStatusExpired OAuthDeviceAuthorizationStatus = "expired"
	OAuthDeviceAuthorizationStatusReady   OAuthDeviceAuthorizationStatus = "approved"
)

type OAuthDeviceAuthorizationDetails struct {
	ID         uuid.UUID
	ClientID   string
	UserCode   string
	Audience   string
	ApprovedBy *uuid.UUID
	ExpiresAt  time.Time
	DeniedAt   *time.Time
	ConsumedAt *time.Time
}

func (s *Service) CreateOAuthDeviceAuthorization(ctx context.Context, req OAuthCreateDeviceAuthorizationRequest) (*OAuthCreateDeviceAuthorizationResponse, error) {
	if strings.TrimSpace(req.ClientID) == "" || len(req.Scopes) == 0 {
		return nil, ErrOAuthInvalidRequest
	}

	deviceCode, err := randomURLSafeToken(40)
	if err != nil {
		return nil, fmt.Errorf("generate device_code: %w", err)
	}
	deviceCodeHash := hashSHA256Hex(deviceCode)
	clientID := strings.TrimSpace(req.ClientID)
	audience := strings.TrimSpace(req.Audience)
	expiresAt := time.Now().Add(OAuthDeviceCodeTTL)

	var userCode string
	for range 5 {
		candidate, codeErr := randomUserCode(8)
		if codeErr != nil {
			return nil, fmt.Errorf("generate user_code: %w", codeErr)
		}
		_, execErr := s.queries.CreateOAuthDeviceAuthorization(ctx, db.CreateOAuthDeviceAuthorizationParams{
			OauthDeviceAuthorizationDeviceCodeHash: &deviceCodeHash,
			OauthClientID:                          &clientID,
			OauthDeviceAuthorizationUserCode:       &candidate,
			OauthDeviceAuthorizationScopes:         req.Scopes,
			OauthDeviceAuthorizationAudience:       &audience,
			OauthDeviceAuthorizationExpiresAt:      &expiresAt,
		})
		if execErr == nil {
			userCode = candidate
			break
		}
		if !isUniqueViolation(execErr) {
			return nil, fmt.Errorf("create oauth device authorization: %w", execErr)
		}
	}
	if userCode == "" {
		return nil, fmt.Errorf("create oauth device authorization: failed to generate unique user_code")
	}

	return &OAuthCreateDeviceAuthorizationResponse{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ExpiresIn:  int(time.Until(expiresAt).Seconds()),
		Interval:   OAuthDevicePollInterval,
	}, nil
}

func (s *Service) ApproveOAuthDeviceAuthorization(ctx context.Context, req OAuthApproveDeviceAuthorizationRequest) error {
	if strings.TrimSpace(req.UserCode) == "" || req.UserID == uuid.Nil {
		return ErrOAuthInvalidRequest
	}
	userCode := strings.ToUpper(strings.TrimSpace(req.UserCode))

	row, err := s.queries.GetOAuthDeviceAuthorizationByUserCode(ctx, &userCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthInvalidRequest
		}
		return fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return ErrOAuthExpiredToken
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil || row.OauthDeviceAuthorizationConsumedAt != nil {
		return ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationApprovedAt != nil {
		return nil
	}

	_, err = s.queries.ApproveOAuthDeviceAuthorizationByUserCode(ctx, db.ApproveOAuthDeviceAuthorizationByUserCodeParams{
		UserUuid:                         &req.UserID,
		OauthDeviceAuthorizationUserCode: &userCode,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthInvalidGrant
		}
		return fmt.Errorf("approve oauth device authorization: %w", err)
	}
	return nil
}

func (s *Service) DenyOAuthDeviceAuthorization(ctx context.Context, req OAuthDenyDeviceAuthorizationRequest) error {
	if strings.TrimSpace(req.UserCode) == "" || req.UserID == uuid.Nil {
		return ErrOAuthInvalidRequest
	}
	userCode := strings.ToUpper(strings.TrimSpace(req.UserCode))
	row, err := s.queries.GetOAuthDeviceAuthorizationByUserCode(ctx, &userCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthInvalidRequest
		}
		return fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return ErrOAuthExpiredToken
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil || row.OauthDeviceAuthorizationConsumedAt != nil {
		return ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationApprovedAt != nil {
		if row.UserUuid == nil || *row.UserUuid != req.UserID {
			return ErrOAuthInvalidGrant
		}
	}
	_, err = s.queries.DenyOAuthDeviceAuthorizationByUserCode(ctx, &userCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOAuthInvalidGrant
		}
		return fmt.Errorf("deny oauth device authorization: %w", err)
	}
	return nil
}

func (s *Service) GetOAuthDeviceAuthorizationDetailsByUserCode(ctx context.Context, userCode string) (*OAuthDeviceAuthorizationDetails, error) {
	trimmedCode := strings.ToUpper(strings.TrimSpace(userCode))
	if trimmedCode == "" {
		return nil, ErrOAuthInvalidRequest
	}
	row, err := s.queries.GetOAuthDeviceAuthorizationByUserCode(ctx, &trimmedCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthInvalidRequest
		}
		return nil, fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	var approvedBy *uuid.UUID
	if row.UserUuid != nil {
		approvedBy = row.UserUuid
	}
	var deniedAt *time.Time
	if row.OauthDeviceAuthorizationDeniedAt != nil {
		deniedAt = row.OauthDeviceAuthorizationDeniedAt
	}
	var consumedAt *time.Time
	if row.OauthDeviceAuthorizationConsumedAt != nil {
		consumedAt = row.OauthDeviceAuthorizationConsumedAt
	}
	return &OAuthDeviceAuthorizationDetails{
		ID:         row.OauthDeviceAuthorizationID,
		ClientID:   row.OauthClientID,
		UserCode:   row.OauthDeviceAuthorizationUserCode,
		Audience:   row.OauthDeviceAuthorizationAudience,
		ApprovedBy: approvedBy,
		ExpiresAt:  row.OauthDeviceAuthorizationExpiresAt,
		DeniedAt:   deniedAt,
		ConsumedAt: consumedAt,
	}, nil
}

func (s *Service) GetOAuthDeviceAuthorizationStatus(ctx context.Context, clientID, deviceCode string) (OAuthDeviceAuthorizationStatus, error) {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(deviceCode) == "" {
		return OAuthDeviceAuthorizationStatusPending, ErrOAuthInvalidRequest
	}
	deviceCodeHash := hashSHA256Hex(strings.TrimSpace(deviceCode))
	row, err := s.queries.GetOAuthDeviceAuthorizationByDeviceCodeHash(ctx, &deviceCodeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OAuthDeviceAuthorizationStatusPending, ErrOAuthInvalidGrant
		}
		return OAuthDeviceAuthorizationStatusPending, fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthClientID != strings.TrimSpace(clientID) || row.OauthDeviceAuthorizationConsumedAt != nil {
		return OAuthDeviceAuthorizationStatusPending, ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return OAuthDeviceAuthorizationStatusExpired, nil
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil {
		return OAuthDeviceAuthorizationStatusDenied, nil
	}
	if row.OauthDeviceAuthorizationApprovedAt != nil && row.UserUuid != nil {
		return OAuthDeviceAuthorizationStatusReady, nil
	}
	return OAuthDeviceAuthorizationStatusPending, nil
}

func (s *Service) ExchangeOAuthDeviceCode(ctx context.Context, req OAuthExchangeDeviceCodeRequest) (*OAuthTokenResponse, error) {
	if strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.DeviceCode) == "" {
		return nil, ErrOAuthInvalidRequest
	}
	deviceCodeHash := hashSHA256Hex(strings.TrimSpace(req.DeviceCode))
	clientID := strings.TrimSpace(req.ClientID)

	row, err := s.queries.GetOAuthDeviceAuthorizationByDeviceCodeHash(ctx, &deviceCodeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthInvalidGrant
		}
		return nil, fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthClientID != clientID || row.OauthDeviceAuthorizationConsumedAt != nil {
		return nil, ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationAudience != "" && strings.TrimSpace(req.Audience) != "" && row.OauthDeviceAuthorizationAudience != strings.TrimSpace(req.Audience) {
		return nil, ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return nil, ErrOAuthExpiredToken
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil {
		return nil, ErrOAuthAccessDenied
	}
	if row.OauthDeviceAuthorizationApprovedAt == nil || row.UserUuid == nil {
		return nil, ErrOAuthPending
	}

	_, err = s.queries.ConsumeOAuthDeviceAuthorizationByID(ctx, &row.OauthDeviceAuthorizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthInvalidGrant
		}
		return nil, fmt.Errorf("consume oauth device authorization: %w", err)
	}

	return s.IssueOAuthTokensForUser(ctx, OAuthIssueTokensForUserRequest{
		ClientID: clientID,
		UserID:   *row.UserUuid,
		Scopes:   row.OauthDeviceAuthorizationScopes,
		Audience: strings.TrimSpace(req.Audience),
	})
}

func (s *Service) ConsumeApprovedOAuthDeviceCode(ctx context.Context, req OAuthConsumeApprovedDeviceCodeRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.DeviceCode) == "" {
		return uuid.Nil, ErrOAuthInvalidRequest
	}
	deviceCodeHash := hashSHA256Hex(strings.TrimSpace(req.DeviceCode))
	clientID := strings.TrimSpace(req.ClientID)
	row, err := s.queries.GetOAuthDeviceAuthorizationByDeviceCodeHash(ctx, &deviceCodeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrOAuthInvalidGrant
		}
		return uuid.Nil, fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthClientID != clientID || row.OauthDeviceAuthorizationConsumedAt != nil {
		return uuid.Nil, ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return uuid.Nil, ErrOAuthExpiredToken
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil {
		return uuid.Nil, ErrOAuthAccessDenied
	}
	if row.OauthDeviceAuthorizationApprovedAt == nil || row.UserUuid == nil {
		return uuid.Nil, ErrOAuthPending
	}

	_, err = s.queries.ConsumeOAuthDeviceAuthorizationByID(ctx, &row.OauthDeviceAuthorizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrOAuthInvalidGrant
		}
		return uuid.Nil, fmt.Errorf("consume oauth device authorization: %w", err)
	}
	return *row.UserUuid, nil
}

type OAuthCreateAuthorizationCodeFromApprovedDeviceRequest struct {
	UserCode            string
	UserID              uuid.UUID
	ClientID            string
	RedirectURI         string
	Scopes              []string
	Audience            string
	CodeChallenge       string
	CodeChallengeMethod string
}

func (s *Service) CreateOAuthAuthorizationCodeFromApprovedDevice(ctx context.Context, req OAuthCreateAuthorizationCodeFromApprovedDeviceRequest) (string, error) {
	userCode := strings.ToUpper(strings.TrimSpace(req.UserCode))
	clientID := strings.TrimSpace(req.ClientID)
	redirectURI := strings.TrimSpace(req.RedirectURI)
	codeChallenge := strings.TrimSpace(req.CodeChallenge)
	codeChallengeMethod := strings.TrimSpace(req.CodeChallengeMethod)
	if userCode == "" || req.UserID == uuid.Nil || clientID == "" || redirectURI == "" || codeChallenge == "" {
		return "", ErrOAuthInvalidRequest
	}
	if !strings.EqualFold(codeChallengeMethod, "S256") || len(req.Scopes) == 0 {
		return "", ErrOAuthInvalidRequest
	}
	if s.pool == nil || s.queries == nil {
		return "", fmt.Errorf("oauth storage is unavailable")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)

	qtx := s.queries.WithTx(tx)
	row, err := qtx.GetOAuthDeviceAuthorizationByUserCode(ctx, &userCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrOAuthInvalidRequest
		}
		return "", fmt.Errorf("lookup oauth device authorization: %w", err)
	}
	if row.OauthClientID != clientID {
		return "", ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationAudience != "" && strings.TrimSpace(req.Audience) != "" && row.OauthDeviceAuthorizationAudience != strings.TrimSpace(req.Audience) {
		return "", ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationExpiresAt.Before(time.Now()) {
		return "", ErrOAuthExpiredToken
	}
	if row.OauthDeviceAuthorizationDeniedAt != nil || row.OauthDeviceAuthorizationConsumedAt != nil {
		return "", ErrOAuthInvalidGrant
	}
	if row.OauthDeviceAuthorizationApprovedAt == nil || row.UserUuid == nil || *row.UserUuid != req.UserID {
		return "", ErrOAuthInvalidGrant
	}

	if _, err := qtx.ConsumeOAuthDeviceAuthorizationByID(ctx, &row.OauthDeviceAuthorizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrOAuthInvalidGrant
		}
		return "", fmt.Errorf("consume oauth device authorization: %w", err)
	}

	code, err := randomURLSafeToken(32)
	if err != nil {
		return "", fmt.Errorf("generate oauth authorization code: %w", err)
	}
	codeHash := hashSHA256Hex(code)
	audience := strings.TrimSpace(req.Audience)
	if _, err := qtx.CreateOAuthAuthorizationCode(ctx, db.CreateOAuthAuthorizationCodeParams{
		OauthAuthorizationCodeCodeHash:            &codeHash,
		OauthClientID:                             &clientID,
		UserUuid:                                  &req.UserID,
		OauthAuthorizationCodeRedirectUri:         &redirectURI,
		OauthAuthorizationCodeScopes:              req.Scopes,
		OauthAuthorizationCodeAudience:            &audience,
		OauthAuthorizationCodeCodeChallenge:       &codeChallenge,
		OauthAuthorizationCodeCodeChallengeMethod: &codeChallengeMethod,
		OauthAuthorizationCodeExpiresAt:           ptr(time.Now().Add(OAuthAuthorizationCodeTTL)),
	}); err != nil {
		return "", fmt.Errorf("persist oauth authorization code: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}
	return code, nil
}

func randomUserCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
