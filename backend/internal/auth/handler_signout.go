package auth

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type SignOutInput struct {
	SessionID string `header:"X-Session-ID" doc:"Session ID to revoke (uses current session if not provided)"`
}

type SignOutOutput struct {
	Body struct {
		Success bool `json:"success"`
	}
}

func (h *Handlers) SignOut(ctx context.Context, input *SignOutInput) (*SignOutOutput, error) {
	userIDStr := GetUserIDFromContext(ctx)
	if userIDStr == "" {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid user")
	}
	var sessionID uuid.UUID
	if input.SessionID != "" {
		parsed, err := uuid.Parse(input.SessionID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid session ID")
		}
		sessionID = parsed
	} else {
		sessionIDStr := GetSessionIDFromContext(ctx)
		if sessionIDStr == "" {
			return nil, huma.Error401Unauthorized("session not found")
		}
		parsed, err := uuid.Parse(sessionIDStr)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid session")
		}
		sessionID = parsed
	}
	if err := h.service.SignOutWithOwnershipCheck(ctx, userID, sessionID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, huma.Error404NotFound("session not found")
		}
		if errors.Is(err, ErrSessionNotOwned) {
			return nil, huma.Error403Forbidden("cannot sign out other users' sessions")
		}
		return nil, huma.Error500InternalServerError("sign out failed", err)
	}
	return &SignOutOutput{
		Body: struct {
			Success bool `json:"success"`
		}{
			Success: true,
		},
	}, nil
}
