package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	db "koditon/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIssueOAuthTokensForUser_BindsAccessTokenToSession(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := newOAuthTestService(t, pool, queries)
	userID := createAuthTestUser(t, ctx, pool, queries)
	sessionID := createOAuthTestSession(t, ctx, service, userID)

	tokenResp, err := service.IssueOAuthTokensForUser(ctx, OAuthIssueTokensForUserRequest{
		ClientID:  "koditon-apple",
		UserID:    userID,
		Scopes:    []string{ScopeProfileRead},
		SessionID: sessionID,
		Audience:  "https://api.example.test",
	})
	if err != nil {
		t.Fatalf("issue oauth tokens: %v", err)
	}

	claims, err := service.VerifyAccessToken(ctx, tokenResp.AccessToken)
	if err != nil {
		t.Fatalf("verify bound access token: %v", err)
	}
	if claims.SessionID != sessionID {
		t.Fatalf("expected session %s, got %s", sessionID, claims.SessionID)
	}

	if err := service.SignOut(ctx, sessionID); err != nil {
		t.Fatalf("sign out session: %v", err)
	}
	if _, err := service.VerifyAccessToken(ctx, tokenResp.AccessToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("expected ErrSessionRevoked after sign-out, got %v", err)
	}
}

func TestRefreshOAuthTokens_ReplayedRefreshTokenRevokesSession(t *testing.T) {
	t.Parallel()

	ctx, pool := openAuthTestPool(t)
	queries := db.New(pool)
	service := newOAuthTestService(t, pool, queries)
	userID := createAuthTestUser(t, ctx, pool, queries)
	sessionID := createOAuthTestSession(t, ctx, service, userID)

	tokenResp, err := service.IssueOAuthTokensForUser(ctx, OAuthIssueTokensForUserRequest{
		ClientID:  "koditon-apple",
		UserID:    userID,
		Scopes:    []string{ScopeProfileRead},
		SessionID: sessionID,
		Audience:  "https://api.example.test",
	})
	if err != nil {
		t.Fatalf("issue oauth tokens: %v", err)
	}

	rotated, err := service.RefreshOAuthTokens(ctx, OAuthRefreshTokensRequest{
		ClientID:     "koditon-apple",
		RefreshToken: tokenResp.RefreshToken,
		Audience:     "https://api.example.test",
	})
	if err != nil {
		t.Fatalf("refresh oauth tokens: %v", err)
	}
	if rotated.RefreshToken == "" || rotated.RefreshToken == tokenResp.RefreshToken {
		t.Fatal("expected a rotated refresh token")
	}

	if _, err := service.RefreshOAuthTokens(ctx, OAuthRefreshTokensRequest{
		ClientID:     "koditon-apple",
		RefreshToken: tokenResp.RefreshToken,
		Audience:     "https://api.example.test",
	}); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse on replay, got %v", err)
	}

	session, err := queries.GetSessionByID(ctx, sessionID)
	if err != nil {
		t.Fatalf("load session after replay: %v", err)
	}
	if session.DeviceSessionRevokedAt == nil {
		t.Fatal("expected session to be revoked after refresh token replay")
	}

	if _, err := service.VerifyAccessToken(ctx, rotated.AccessToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("expected ErrSessionRevoked for rotated access token after replay, got %v", err)
	}
}

func newOAuthTestService(t *testing.T, pool *pgxpool.Pool, queries *db.Queries) *Service {
	t.Helper()

	jwtService, err := NewJWTService(JWTConfig{
		Issuer:      "koditon-test",
		UIDHashSalt: "oauth-flow-test-salt",
	})
	if err != nil {
		t.Fatalf("create jwt service: %v", err)
	}

	return &Service{
		logger:     slog.Default(),
		pool:       pool,
		queries:    queries,
		jwtService: jwtService,
		policy:     defaultPolicy(),
	}
}

func createOAuthTestSession(t *testing.T, ctx context.Context, service *Service, userID uuid.UUID) uuid.UUID {
	t.Helper()

	tx, err := service.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer rollbackTx(ctx, slog.Default(), tx)

	sessionID, err := service.createSessionWithProvider(ctx, tx, createSessionParams{
		UserID:     userID,
		Provider:   AuthProviderEmail,
		DeviceID:   uuid.New(),
		DeviceName: "OAuth Flow Test Device",
		DeviceOS:   "iOS 26",
		IP:         "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit session transaction: %v", err)
	}
	return sessionID
}
