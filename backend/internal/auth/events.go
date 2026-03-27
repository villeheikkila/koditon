package auth

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
)

const (
	tokenEventIssued        = "token_issued"
	tokenEventRotated       = "token_rotated"
	tokenEventReuseDetected = "token_reuse_detected"
	tokenEventRevoked       = "token_revoked"
)

type tokenEvent struct {
	Name      string
	AuthType  string
	ClientID  string
	SessionID uuid.UUID
	UserID    uuid.UUID
	Scopes    []string
	TokenType string
}

func (s *Service) emitTokenEvent(ctx context.Context, evt tokenEvent) {
	if s == nil || s.logger == nil || strings.TrimSpace(evt.Name) == "" {
		return
	}
	scopeSet := append([]string(nil), evt.Scopes...)
	slices.Sort(scopeSet)
	s.logger.InfoContext(ctx, "auth token event",
		"event", evt.Name,
		"auth_type", strings.TrimSpace(evt.AuthType),
		"client_id", strings.TrimSpace(evt.ClientID),
		"session_id", evt.SessionID,
		"user_id", evt.UserID,
		"scope_set", scopeSet,
		"token_type", strings.TrimSpace(evt.TokenType),
	)
}
