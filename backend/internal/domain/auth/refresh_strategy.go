package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "koditon/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type refreshFlowState struct {
	UserID           uuid.UUID
	SessionID        uuid.UUID
	Provider         string
	ClientID         string
	Scopes           []string
	Audience         string
	RefreshCounter   int64
	SessionNotAfter  time.Time
	NextRefreshToken string
	RotatedTokenID   uuid.UUID
}

type refreshValidation struct {
	state         refreshFlowState
	reuseDetected bool
}

type refreshStrategy interface {
	validate(ctx context.Context, tx pgx.Tx, queries *db.Queries) (refreshValidation, error)
	handleReuse(ctx context.Context, tx pgx.Tx, queries *db.Queries, state refreshFlowState) error
	rotate(ctx context.Context, tx pgx.Tx, queries *db.Queries, state refreshFlowState) (refreshFlowState, error)
}

func refreshTokenStateFromLockedRow(row db.GetOAuthRefreshTokenByHashForUpdateRow, audience string) refreshFlowState {
	return refreshFlowState{
		UserID:         row.UserUuid,
		SessionID:      uuidValue(row.DeviceSessionUuid),
		ClientID:       row.OauthClientID,
		Scopes:         row.OauthRefreshTokenScopes,
		Audience:       audience,
		RotatedTokenID: row.OauthRefreshTokenID,
	}
}

func (s *Service) runRefreshFlow(ctx context.Context, strategy refreshStrategy) (refreshFlowState, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return refreshFlowState{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
	qtx := s.queries.WithTx(tx)

	validation, err := strategy.validate(ctx, tx, qtx)
	if err != nil {
		return refreshFlowState{}, err
	}
	if validation.reuseDetected {
		validation.state = strategyEventDefaults(validation.state)
		s.emitTokenEvent(ctx, tokenEvent{
			Name:      tokenEventReuseDetected,
			AuthType:  strategyAuthType(validation.state),
			ClientID:  validation.state.ClientID,
			SessionID: validation.state.SessionID,
			UserID:    validation.state.UserID,
			Scopes:    validation.state.Scopes,
			TokenType: "refresh",
		})
		if err := strategy.handleReuse(ctx, tx, qtx, validation.state); err != nil {
			return refreshFlowState{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return refreshFlowState{}, fmt.Errorf("commit refresh reuse handling: %w", err)
		}
		return refreshFlowState{}, ErrTokenReuse
	}

	rotated, err := strategy.rotate(ctx, tx, qtx, validation.state)
	if err != nil {
		return refreshFlowState{}, err
	}
	rotated = strategyEventDefaults(rotated)
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRotated,
		AuthType:  strategyAuthType(rotated),
		ClientID:  rotated.ClientID,
		SessionID: rotated.SessionID,
		UserID:    rotated.UserID,
		Scopes:    rotated.Scopes,
		TokenType: "refresh",
	})
	if err := tx.Commit(ctx); err != nil {
		return refreshFlowState{}, fmt.Errorf("commit refresh transaction: %w", err)
	}
	return rotated, nil
}

type oauthOpaqueRefreshStrategy struct {
	service      *Service
	clientID     string
	refreshToken string
	audience     string
}

func (s oauthOpaqueRefreshStrategy) validate(ctx context.Context, _ pgx.Tx, queries *db.Queries) (refreshValidation, error) {
	row, err := queries.GetOAuthRefreshTokenByHashForUpdate(ctx, hashSHA256Hex(s.refreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return refreshValidation{}, ErrOAuthInvalidGrant
		}
		return refreshValidation{}, fmt.Errorf("get oauth refresh token: %w", err)
	}
	if row.OauthClientID != s.clientID {
		return refreshValidation{}, ErrOAuthInvalidGrant
	}
	storedAudience := row.OauthRefreshTokenAudience
	requestedAudience := s.audience
	if storedAudience != "" && requestedAudience != "" && storedAudience != requestedAudience {
		return refreshValidation{}, ErrOAuthInvalidGrant
	}
	if storedAudience == "" {
		storedAudience = requestedAudience
	}
	state := refreshTokenStateFromLockedRow(row, storedAudience)
	if row.OauthRefreshTokenRevokedAt != nil {
		return refreshValidation{
			state:         state,
			reuseDetected: true,
		}, nil
	}
	if !row.OauthRefreshTokenExpiresAt.After(time.Now()) {
		return refreshValidation{}, ErrOAuthInvalidGrant
	}
	revoked, err := queries.RevokeOAuthRefreshTokenByHash(ctx, hashSHA256Hex(s.refreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return refreshValidation{}, ErrOAuthInvalidGrant
		}
		return refreshValidation{}, fmt.Errorf("revoke oauth refresh token: %w", err)
	}
	state.RotatedTokenID = revoked.OauthRefreshTokenID
	state.Scopes = revoked.OauthRefreshTokenScopes
	return refreshValidation{
		state: state,
	}, nil
}

func (s oauthOpaqueRefreshStrategy) handleReuse(ctx context.Context, _ pgx.Tx, queries *db.Queries, state refreshFlowState) error {
	if _, err := queries.RevokeAllOAuthRefreshTokensByUserIDAndClientID(ctx, db.RevokeAllOAuthRefreshTokensByUserIDAndClientIDParams{
		UserUuid:      state.UserID,
		OauthClientID: state.ClientID,
	}); err != nil {
		return fmt.Errorf("revoke oauth refresh tokens after reuse: %w", err)
	}
	if err := queries.RevokeAllUserSessions(ctx, state.UserID); err != nil {
		return fmt.Errorf("revoke user sessions after refresh token reuse: %w", err)
	}
	return nil
}

func (s oauthOpaqueRefreshStrategy) rotate(ctx context.Context, _ pgx.Tx, queries *db.Queries, state refreshFlowState) (refreshFlowState, error) {
	refreshToken, refreshExpiry, err := createOAuthRefreshToken(ctx, queries, state.ClientID, state.UserID, state.SessionID, state.Scopes, state.Audience, state.RotatedTokenID, s.service.policy.OAuthRefreshTokenTTL)
	if err != nil {
		return refreshFlowState{}, fmt.Errorf("rotate oauth refresh token: %w", err)
	}
	state.NextRefreshToken = refreshToken
	state.SessionNotAfter = refreshExpiry
	return state, nil
}

func uuidValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func strategyAuthType(state refreshFlowState) string {
	if state.ClientID != "" {
		return string(AccessTokenKindOAuth)
	}
	return state.Provider
}

func strategyEventDefaults(state refreshFlowState) refreshFlowState {
	if state.ClientID != "" {
		return state
	}
	if state.Provider == "" {
		state.Provider = string(AccessTokenKindApp)
	}
	return state
}
