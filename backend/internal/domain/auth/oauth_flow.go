package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	db "koditon/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	OAuthAuthorizationCodeTTL = 5 * time.Minute
)

type OAuthCreateAuthorizationCodeRequest struct {
	ClientID            string
	UserID              uuid.UUID
	RedirectURI         string
	Scopes              []string
	Audience            string
	CodeChallenge       string
	CodeChallengeMethod string
}

type OAuthExchangeAuthorizationCodeRequest struct {
	ClientID     string
	Code         string
	RedirectURI  string
	CodeVerifier string
	Audience     string
}

type OAuthRefreshTokensRequest struct {
	ClientID     string
	RefreshToken string
	Audience     string
}

type OAuthTokenResponse struct {
	AccessToken   string
	AccessExpiry  time.Time
	RefreshToken  string
	RefreshExpiry time.Time
	Scopes        []string
	UserID        uuid.UUID
	SessionID     uuid.UUID
}

type OAuthIssueTokensForUserRequest struct {
	ClientID  string
	UserID    uuid.UUID
	Scopes    []string
	SessionID uuid.UUID
	Audience  string
}

func (s *Service) CreateOAuthAuthorizationCode(ctx context.Context, req OAuthCreateAuthorizationCodeRequest) (string, error) {
	if strings.TrimSpace(req.ClientID) == "" || req.UserID == uuid.Nil || strings.TrimSpace(req.RedirectURI) == "" {
		return "", ErrOAuthInvalidRequest
	}
	if strings.TrimSpace(req.CodeChallenge) == "" || !strings.EqualFold(strings.TrimSpace(req.CodeChallengeMethod), "S256") {
		return "", ErrOAuthInvalidRequest
	}
	if len(req.Scopes) == 0 {
		return "", ErrOAuthInvalidRequest
	}

	code, err := randomURLSafeToken(32)
	if err != nil {
		return "", fmt.Errorf("generate oauth authorization code: %w", err)
	}
	codeHash := hashSHA256Hex(code)
	clientID := strings.TrimSpace(req.ClientID)
	redirectURI := strings.TrimSpace(req.RedirectURI)
	audience := strings.TrimSpace(req.Audience)
	codeChallenge := strings.TrimSpace(req.CodeChallenge)
	codeChallengeMethod := strings.TrimSpace(req.CodeChallengeMethod)
	if _, err := s.queries.CreateOAuthAuthorizationCode(ctx, db.CreateOAuthAuthorizationCodeParams{
		OauthAuthorizationCodeCodeHash:            codeHash,
		OauthClientID:                             clientID,
		UserUuid:                                  req.UserID,
		OauthAuthorizationCodeRedirectUri:         redirectURI,
		OauthAuthorizationCodeScopes:              req.Scopes,
		OauthAuthorizationCodeAudience:            audience,
		OauthAuthorizationCodeCodeChallenge:       codeChallenge,
		OauthAuthorizationCodeCodeChallengeMethod: codeChallengeMethod,
		OauthAuthorizationCodeExpiresAt:           time.Now().Add(OAuthAuthorizationCodeTTL),
	}); err != nil {
		return "", fmt.Errorf("persist oauth authorization code: %w", err)
	}
	return code, nil
}

func (s *Service) ExchangeOAuthAuthorizationCode(ctx context.Context, req OAuthExchangeAuthorizationCodeRequest) (*OAuthTokenResponse, error) {
	clientID := strings.TrimSpace(req.ClientID)
	code := strings.TrimSpace(req.Code)
	redirectURI := strings.TrimSpace(req.RedirectURI)
	codeVerifier := strings.TrimSpace(req.CodeVerifier)
	if clientID == "" || code == "" || redirectURI == "" || codeVerifier == "" {
		return nil, ErrOAuthInvalidRequest
	}
	codeChallenge := pkceS256Challenge(codeVerifier)
	codeHash := hashSHA256Hex(code)
	audience := strings.TrimSpace(req.Audience)
	codeChallengeMethod := "S256"
	row, err := s.queries.ConsumeOAuthAuthorizationCode(ctx, db.ConsumeOAuthAuthorizationCodeParams{
		OauthAuthorizationCodeCodeHash:            codeHash,
		OauthClientID:                             clientID,
		OauthAuthorizationCodeRedirectUri:         redirectURI,
		OauthAuthorizationCodeAudience:            audience,
		OauthAuthorizationCodeCodeChallenge:       codeChallenge,
		OauthAuthorizationCodeCodeChallengeMethod: codeChallengeMethod,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthInvalidGrant
		}
		return nil, fmt.Errorf("consume oauth authorization code: %w", err)
	}

	accessToken, accessExpiry, err := s.IssueAccessToken(ctx, IssueAccessTokenRequest{
		UserID:    row.UserUuid,
		Scopes:    row.OauthAuthorizationCodeScopes,
		SessionID: uuid.Nil,
		ClientID:  row.OauthClientID,
		Audience:  strings.TrimSpace(req.Audience),
		TokenKind: AccessTokenKindOAuth,
		TTL:       s.policy.OAuthAccessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create oauth access token: %w", err)
	}
	refreshToken, refreshExpiry, err := createOAuthRefreshToken(ctx, s.queries, row.OauthClientID, row.UserUuid, uuid.Nil, row.OauthAuthorizationCodeScopes, strings.TrimSpace(req.Audience), uuid.Nil, s.policy.OAuthRefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("create oauth refresh token: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventIssued,
		AuthType:  string(AccessTokenKindOAuth),
		ClientID:  row.OauthClientID,
		UserID:    row.UserUuid,
		Scopes:    row.OauthAuthorizationCodeScopes,
		TokenType: "refresh",
	})
	return &OAuthTokenResponse{
		AccessToken:   accessToken,
		AccessExpiry:  accessExpiry,
		RefreshToken:  refreshToken,
		RefreshExpiry: refreshExpiry,
		Scopes:        row.OauthAuthorizationCodeScopes,
		UserID:        row.UserUuid,
		SessionID:     uuid.Nil,
	}, nil
}

func (s *Service) RefreshOAuthTokens(ctx context.Context, req OAuthRefreshTokensRequest) (*OAuthTokenResponse, error) {
	if strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.RefreshToken) == "" {
		return nil, ErrOAuthInvalidRequest
	}
	rotated, err := s.runRefreshFlow(ctx, oauthOpaqueRefreshStrategy{
		service:      s,
		clientID:     strings.TrimSpace(req.ClientID),
		refreshToken: strings.TrimSpace(req.RefreshToken),
		audience:     strings.TrimSpace(req.Audience),
	})
	if err != nil {
		return nil, err
	}

	accessToken, accessExpiry, err := s.IssueAccessToken(ctx, IssueAccessTokenRequest{
		UserID:    rotated.UserID,
		Scopes:    rotated.Scopes,
		SessionID: uuid.Nil,
		ClientID:  rotated.ClientID,
		Audience:  strings.TrimSpace(req.Audience),
		TokenKind: AccessTokenKindOAuth,
		TTL:       s.policy.OAuthAccessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create oauth access token: %w", err)
	}
	return &OAuthTokenResponse{
		AccessToken:   accessToken,
		AccessExpiry:  accessExpiry,
		RefreshToken:  rotated.NextRefreshToken,
		RefreshExpiry: rotated.SessionNotAfter,
		Scopes:        rotated.Scopes,
		UserID:        rotated.UserID,
		SessionID:     rotated.SessionID,
	}, nil
}

func (s *Service) IssueOAuthTokensForUser(ctx context.Context, req OAuthIssueTokensForUserRequest) (*OAuthTokenResponse, error) {
	if strings.TrimSpace(req.ClientID) == "" || req.UserID == uuid.Nil {
		return nil, ErrOAuthInvalidRequest
	}
	if len(req.Scopes) == 0 {
		return nil, ErrOAuthInvalidRequest
	}
	accessToken, accessExpiry, err := s.IssueAccessToken(ctx, IssueAccessTokenRequest{
		UserID:    req.UserID,
		Scopes:    req.Scopes,
		SessionID: req.SessionID,
		ClientID:  strings.TrimSpace(req.ClientID),
		Audience:  strings.TrimSpace(req.Audience),
		TokenKind: AccessTokenKindOAuth,
		TTL:       s.policy.OAuthAccessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create oauth access token: %w", err)
	}
	refreshToken, refreshExpiry, err := createOAuthRefreshToken(ctx, s.queries, strings.TrimSpace(req.ClientID), req.UserID, req.SessionID, req.Scopes, strings.TrimSpace(req.Audience), uuid.Nil, s.policy.OAuthRefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("create oauth refresh token: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventIssued,
		AuthType:  string(AccessTokenKindOAuth),
		ClientID:  strings.TrimSpace(req.ClientID),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Scopes:    req.Scopes,
		TokenType: "refresh",
	})
	return &OAuthTokenResponse{
		AccessToken:   accessToken,
		AccessExpiry:  accessExpiry,
		RefreshToken:  refreshToken,
		RefreshExpiry: refreshExpiry,
		Scopes:        req.Scopes,
		UserID:        req.UserID,
		SessionID:     req.SessionID,
	}, nil
}

func createOAuthRefreshToken(ctx context.Context, queries *db.Queries, clientID string, userID, sessionID uuid.UUID, scopes []string, audience string, rotatedFrom uuid.UUID, ttl time.Duration) (string, time.Time, error) {
	refreshToken, err := randomURLSafeToken(40)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate oauth refresh token: %w", err)
	}
	if ttl <= 0 {
		ttl = defaultOAuthRefreshTokenTTL
	}
	expiresAt := time.Now().Add(ttl)
	var rotatedFromValue *uuid.UUID
	if rotatedFrom != uuid.Nil {
		rotatedFromValue = &rotatedFrom
	}
	var sessionIDValue *uuid.UUID
	if sessionID != uuid.Nil {
		sessionIDValue = &sessionID
	}
	if _, err := queries.CreateOAuthRefreshToken(ctx, db.CreateOAuthRefreshTokenParams{
		OauthRefreshTokenTokenHash:   hashSHA256Hex(refreshToken),
		OauthClientID:                clientID,
		UserUuid:                     userID,
		DeviceSessionUuid:            sessionIDValue,
		OauthRefreshTokenScopes:      scopes,
		OauthRefreshTokenAudience:    strings.TrimSpace(audience),
		OauthRefreshTokenExpiresAt:   expiresAt,
		OauthRefreshTokenRotatedFrom: rotatedFromValue,
	}); err != nil {
		return "", time.Time{}, err
	}
	return refreshToken, expiresAt, nil
}

func pkceS256Challenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLSafeToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("size must be positive")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashSHA256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
