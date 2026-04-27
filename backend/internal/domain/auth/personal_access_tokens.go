package auth

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	db "koditon-go/internal/db"
	"koditon-go/internal/platform/logging"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	apiKeyPrefix         = "mk_"
	apiKeyPrefixBytes    = 6
	apiKeySecretBytes    = 32
	apiKeyPrefixAttempts = 5
)

func (s *Service) CreatePersonalAccessToken(ctx context.Context, userID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (string, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	roles, err := s.GetRoleNamesByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user roles: %w", err)
	}
	allowedScopes := ScopesForRoles(roles)
	if len(scopes) > 0 && !HasScopes(allowedScopes, scopes) {
		return "", fmt.Errorf("token scopes exceed user scopes")
	}
	var scopeList []string
	if len(scopes) > 0 {
		scopeList = scopes
	}
	for range apiKeyPrefixAttempts {
		prefix, secret, err := generatePersonalAccessToken()
		if err != nil {
			return "", fmt.Errorf("generate token: %w", err)
		}
		tokenHash := hashPersonalAccessTokenSecretSHA256(secret)
		_, err = s.queries.CreatePersonalAccessToken(ctx, db.CreatePersonalAccessTokenParams{
			UserID:                       user.UserIDBigint,
			PersonalAccessTokenName:      name,
			PersonalAccessTokenPrefix:    prefix,
			PersonalAccessTokenTokenHash: tokenHash,
			PersonalAccessTokenScopes:    scopeList,
			PersonalAccessTokenExpiresAt: expiresAt,
		})
		if err == nil {
			return fmt.Sprintf("%s.%s", prefix, secret), nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return "", fmt.Errorf("create token: %w", err)
	}
	return "", fmt.Errorf("create token: could not generate unique prefix")
}

func (s *Service) VerifyPersonalAccessToken(ctx context.Context, token string) (*AccessTokenClaims, error) {
	prefix, secret, ok := splitPersonalAccessToken(token)
	if !ok {
		return nil, ErrInvalidToken
	}
	row, err := s.queries.GetPersonalAccessTokenByPrefix(ctx, prefix)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("get token: %w", err)
	}
	if row.PersonalAccessTokenRevokedAt != nil {
		return nil, ErrTokenRevoked
	}
	if row.PersonalAccessTokenExpiresAt != nil && time.Now().After(*row.PersonalAccessTokenExpiresAt) {
		return nil, ErrTokenExpired
	}
	expected := []byte(row.PersonalAccessTokenTokenHash)
	actual, ok := hashPersonalAccessTokenSecret(secret, row.PersonalAccessTokenTokenHash)
	if !ok {
		return nil, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return nil, ErrInvalidToken
	}
	userID := row.UserUuid
	userIDHash, err := s.jwtService.uidHasher.EncodeInt64(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("encode user id: %w", err)
	}
	roles, err := s.GetRoleNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	flags, err := s.GetActiveFeatureFlagsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get feature flags: %w", err)
	}
	if err := s.queries.UpdatePersonalAccessTokenLastUsed(ctx, row.PersonalAccessTokenID); err != nil {
		logging.With(s.logger, logging.Op("auth.personal_access_token.last_used"), slog.String("token_prefix", prefix)).WarnContext(ctx, "personal access token last used update failed", "error", err, "outcome", logging.OutcomeError)
	}
	claims := &AccessTokenClaims{
		UserID:       userID,
		UserIDHash:   userIDHash,
		UserIDInt64:  row.UserID,
		SessionID:    uuid.Nil,
		Roles:        roles,
		FeatureFlags: flags,
		Scopes:       row.PersonalAccessTokenScopes,
		IssuedAt:     time.Now(),
	}
	if row.PersonalAccessTokenExpiresAt != nil {
		claims.ExpiresAt = *row.PersonalAccessTokenExpiresAt
	}
	logging.With(s.logger, logging.Op("auth.personal_access_token.verify"), slog.String("token_prefix", prefix), slog.Any("user_id", userID)).DebugContext(ctx, "personal access token authenticated", "outcome", logging.OutcomeSuccess)
	return claims, nil
}

func generatePersonalAccessToken() (string, string, error) {
	prefixBytes := make([]byte, apiKeyPrefixBytes)
	if _, err := rand.Read(prefixBytes); err != nil {
		return "", "", err
	}
	secretBytes := make([]byte, apiKeySecretBytes)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}
	prefix := apiKeyPrefix + hex.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	return prefix, secret, nil
}

func hashPersonalAccessTokenSecret(secret string, stored string) ([]byte, bool) {
	switch len(stored) {
	case 64:
		sum := sha256.Sum256([]byte(secret))
		return []byte(hex.EncodeToString(sum[:])), true
	case 32:
		sum := md5.Sum([]byte(secret))
		return []byte(hex.EncodeToString(sum[:])), true
	default:
		return nil, false
	}
}

func hashPersonalAccessTokenSecretSHA256(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func splitPersonalAccessToken(token string) (string, string, bool) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, apiKeyPrefix) {
		return "", "", false
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	prefix := parts[0]
	secret := strings.TrimSpace(parts[1])
	if secret == "" {
		return "", "", false
	}
	return prefix, secret, true
}
