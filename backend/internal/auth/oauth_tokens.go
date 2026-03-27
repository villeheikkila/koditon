package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateOAuthAccessToken(ctx context.Context, userID uuid.UUID, scopes []string, audience string, ttl time.Duration) (string, time.Time, error) {
	token, expiresAt, err := s.IssueAccessToken(ctx, IssueAccessTokenRequest{
		UserID:    userID,
		Scopes:    scopes,
		SessionID: uuid.Nil,
		Audience:  audience,
		TokenKind: AccessTokenKindOAuth,
		TTL:       ttl,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("issue oauth access token: %w", err)
	}
	return token, expiresAt, nil
}
