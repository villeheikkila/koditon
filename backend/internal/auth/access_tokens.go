package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AccessTokenKind string

const (
	AccessTokenKindApp   AccessTokenKind = "app"
	AccessTokenKindOAuth AccessTokenKind = "oauth"
)

type IssueAccessTokenRequest struct {
	UserID    uuid.UUID
	Scopes    []string
	SessionID uuid.UUID
	ClientID  string
	Audience  string
	TokenKind AccessTokenKind
	TTL       time.Duration
}

func (s *Service) IssueAccessToken(ctx context.Context, req IssueAccessTokenRequest) (string, time.Time, error) {
	if req.UserID == uuid.Nil {
		return "", time.Time{}, fmt.Errorf("issue access token: user id is required")
	}
	if req.TokenKind == "" {
		req.TokenKind = AccessTokenKindApp
	}
	if req.TTL <= 0 {
		switch req.TokenKind {
		case AccessTokenKindOAuth:
			req.TTL = s.policy.OAuthAccessTokenTTL
		case AccessTokenKindApp:
			req.TTL = s.policy.AppAccessTokenTTL
		default:
			return "", time.Time{}, fmt.Errorf("issue access token: unsupported token kind %q", req.TokenKind)
		}
	}

	user, err := s.GetUserByID(ctx, req.UserID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("get user: %w", err)
	}
	roles, err := s.GetRoleNamesByUserID(ctx, req.UserID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("get user roles: %w", err)
	}
	allowedScopes := ScopesForRoles(roles)
	if len(req.Scopes) > 0 && !HasScopes(allowedScopes, req.Scopes) {
		return "", time.Time{}, fmt.Errorf("requested scopes exceed user scopes")
	}
	flags, err := s.GetActiveFeatureFlagsByUserID(ctx, req.UserID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("get feature flags: %w", err)
	}
	userIDHash, err := s.jwtService.uidHasher.EncodeInt64(user.UserIDBigint)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode user id: %w", err)
	}

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(req.TTL)
	token, err := s.jwtService.SignAccessToken(AccessTokenClaims{
		UserID:       req.UserID,
		UserIDHash:   userIDHash,
		UserIDInt64:  user.UserIDBigint,
		SessionID:    req.SessionID,
		Roles:        roles,
		FeatureFlags: flags,
		Scopes:       req.Scopes,
		Audience:     req.Audience,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventIssued,
		AuthType:  string(req.TokenKind),
		ClientID:  req.ClientID,
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Scopes:    req.Scopes,
		TokenType: "access",
	})
	return token, expiresAt, nil
}
