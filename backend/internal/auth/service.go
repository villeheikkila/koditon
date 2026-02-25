package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"koditon-go/internal/auth/apple"
	"koditon-go/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	logger      *slog.Logger
	pool        *pgxpool.Pool
	queries     *db.Queries
	jwtService  *JWTService
	appleClient *apple.Client
}

type ServiceConfig struct {
	Pool   *pgxpool.Pool
	JWT    JWTConfig
	Apple  *apple.Config
	Logger *slog.Logger
}

func NewService(ctx context.Context, cfg ServiceConfig) (*Service, error) {
	jwtService, err := NewJWTService(cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("create jwt service: %w", err)
	}
	var appleClient *apple.Client
	if cfg.Apple != nil && cfg.Apple.PrivateKey != "" {
		appleClient, err = apple.NewClient(ctx, *cfg.Apple)
		if err != nil {
			return nil, fmt.Errorf("create apple client: %w", err)
		}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		logger:      logger.With("component", "auth"),
		pool:        cfg.Pool,
		queries:     db.New(cfg.Pool),
		jwtService:  jwtService,
		appleClient: appleClient,
	}, nil
}

type SignInAnonymousRequest struct {
	DeviceID  *pgtype.UUID
	UserAgent string
	IP        string
}

type SignInAnonymousResponse struct {
	Tokens    TokenPair
	UserID    uuid.UUID
	IsNewUser bool
}

type SignInWithAppleRequest struct {
	AuthorizationCode string
	Nonce             string
	DeviceID          *pgtype.UUID
	UserAgent         string
	IP                string
}

type SignInWithAppleResponse struct {
	Tokens    TokenPair
	UserID    uuid.UUID
	IsNewUser bool
}

func (s *Service) SignInAnonymous(ctx context.Context, req SignInAnonymousRequest) (*SignInAnonymousResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)
	user, err := qtx.CreateUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	externalID := uuid.New().String()
	_, err = qtx.CreateIdentity(ctx, &db.CreateIdentityParams{
		UserID:             user.UserID,
		IdentityProvider:   db.AuthAuthProviderAnonymous,
		IdentityExternalID: externalID,
		IdentityData:       []byte("{}"),
	})
	if err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}
	hmacKey, err := generateHMACKey()
	if err != nil {
		return nil, fmt.Errorf("generate hmac key: %w", err)
	}
	sessionNotAfter := time.Now().Add(RefreshTokenExpiry)
	var ip *string
	if req.IP != "" {
		ip = &req.IP
	}
	var userAgent *string
	if req.UserAgent != "" {
		userAgent = &req.UserAgent
	}
	var deviceID pgtype.UUID
	if req.DeviceID != nil {
		deviceID = *req.DeviceID
		_, err = qtx.UpsertDevice(ctx, &db.UpsertDeviceParams{
			DeviceID: deviceID,
			UserID:   user.UserID,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert device: %w", err)
		}
	}
	session, err := qtx.CreateSession(ctx, &db.CreateSessionParams{
		UserID:                     user.UserID,
		SessionDeviceID:            deviceID,
		SessionUserAgent:           userAgent,
		SessionIp:                  ip,
		SessionProvider:            db.AuthAuthProviderAnonymous,
		SessionRefreshTokenHmacKey: hmacKey,
		SessionNotAfter:            &sessionNotAfter,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	tokenHash := hashToken(hmacKey, session.SessionRefreshTokenCounter)
	_, err = qtx.CreateRefreshToken(ctx, &db.CreateRefreshTokenParams{
		SessionID:             session.SessionID,
		RefreshTokenTokenHash: tokenHash,
		RefreshTokenCounter:   session.SessionRefreshTokenCounter,
	})
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	sessionUUID := pgToUUID(session.SessionID)
	userUUID := pgToUUID(user.UserID)
	tokens, err := s.generateTokenPair(ctx, sessionUUID, userUUID, session.SessionRefreshTokenCounter, sessionNotAfter, "", 0)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	s.logger.Info("new anonymous user created", "user_id", userUUID)
	return &SignInAnonymousResponse{
		Tokens:    tokens,
		UserID:    userUUID,
		IsNewUser: true,
	}, nil
}

func (s *Service) SignInWithApple(ctx context.Context, req SignInWithAppleRequest) (*SignInWithAppleResponse, error) {
	s.logger.Debug("apple sign in started",
		"has_auth_code", req.AuthorizationCode != "",
		"has_nonce", req.Nonce != "",
		"has_device_id", req.DeviceID != nil,
		"user_agent", req.UserAgent,
	)
	if s.appleClient == nil {
		s.logger.Error("apple client not configured")
		return nil, errors.New("apple client not configured")
	}
	s.logger.Debug("exchanging authorization code with apple")
	tokenResp, err := s.appleClient.ExchangeAuthorizationCode(ctx, req.AuthorizationCode)
	if err != nil {
		s.logger.Error("failed to exchange authorization code", "error", err)
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	s.logger.Debug("authorization code exchanged successfully",
		"has_id_token", tokenResp.IDToken != "",
		"has_access_token", tokenResp.AccessToken != "",
		"has_refresh_token", tokenResp.RefreshToken != "",
	)
	if tokenResp.IDToken == "" {
		s.logger.Error("apple returned empty id token")
		return nil, ErrMissingIDToken
	}
	s.logger.Debug("verifying apple id token")
	identity, err := s.appleClient.VerifyIDToken(ctx, tokenResp.IDToken, req.Nonce)
	if err != nil {
		s.logger.Error("failed to verify id token", "error", err)
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	s.logger.Debug("id token verified",
		"subject", identity.Subject,
		"email", identity.Email,
		"email_verified", identity.EmailVerified,
	)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)
	var email *string
	if identity.Email != "" {
		email = &identity.Email
	}
	emailVerified := identity.EmailVerified == "true"
	identityData, _ := json.Marshal(map[string]any{
		"iss":            apple.Issuer(),
		"sub":            identity.Subject,
		"email":          identity.Email,
		"provider_id":    identity.Subject,
		"custom_claims":  map[string]any{"auth_time": identity.AuthTime},
		"email_verified": emailVerified,
		"phone_verified": false,
	})
	s.logger.Debug("looking up existing identity", "provider", "apple", "external_id", identity.Subject)
	existingIdentity, err := qtx.GetIdentityByProviderAndExternalID(ctx, &db.GetIdentityByProviderAndExternalIDParams{
		IdentityProvider:   db.AuthAuthProviderApple,
		IdentityExternalID: identity.Subject,
	})
	var userID pgtype.UUID
	var isNewUser bool
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("failed to lookup identity", "error", err)
			return nil, fmt.Errorf("get identity: %w", err)
		}
		s.logger.Debug("no existing identity found, creating new user")
		user, createErr := qtx.CreateUser(ctx)
		if createErr != nil {
			s.logger.Error("failed to create user", "error", createErr)
			return nil, fmt.Errorf("create user: %w", createErr)
		}
		userID = user.UserID
		isNewUser = true
		s.logger.Debug("creating identity for new user", "user_id", pgToUUID(userID))
		_, err = qtx.CreateIdentity(ctx, &db.CreateIdentityParams{
			UserID:                userID,
			IdentityProvider:      db.AuthAuthProviderApple,
			IdentityExternalID:    identity.Subject,
			IdentityEmail:         email,
			IdentityEmailVerified: pgtype.Bool{Bool: emailVerified, Valid: true},
			IdentityData:          identityData,
		})
		if err != nil {
			s.logger.Error("failed to create identity", "error", err)
			return nil, fmt.Errorf("create identity: %w", err)
		}
		s.logger.Info("new user created via apple sign in", "user_id", pgToUUID(userID))
	} else {
		userID = existingIdentity.UserID
		s.logger.Debug("existing user found", "user_id", pgToUUID(userID), "identity_id", pgToUUID(existingIdentity.IdentityID))
		_, err = qtx.UpdateIdentity(ctx, &db.UpdateIdentityParams{
			IdentityID:            existingIdentity.IdentityID,
			IdentityEmail:         email,
			IdentityEmailVerified: pgtype.Bool{Bool: emailVerified, Valid: true},
			IdentityData:          identityData,
		})
		if err != nil {
			s.logger.Error("failed to update identity", "error", err)
			return nil, fmt.Errorf("update identity: %w", err)
		}
		s.logger.Debug("identity updated successfully")
	}
	hmacKey, err := generateHMACKey()
	if err != nil {
		return nil, fmt.Errorf("generate hmac key: %w", err)
	}
	sessionNotAfter := time.Now().Add(RefreshTokenExpiry)
	var ip *string
	if req.IP != "" {
		ip = &req.IP
	}
	var userAgent *string
	if req.UserAgent != "" {
		userAgent = &req.UserAgent
	}
	var deviceID pgtype.UUID
	if req.DeviceID != nil {
		deviceID = *req.DeviceID
		s.logger.Debug("upserting device", "device_id", pgToUUID(deviceID), "user_id", pgToUUID(userID))
		_, err = qtx.UpsertDevice(ctx, &db.UpsertDeviceParams{
			DeviceID: deviceID,
			UserID:   userID,
		})
		if err != nil {
			s.logger.Error("failed to upsert device", "error", err)
			return nil, fmt.Errorf("upsert device: %w", err)
		}
	}
	s.logger.Debug("creating session", "user_id", pgToUUID(userID), "has_device_id", deviceID.Valid)
	session, err := qtx.CreateSession(ctx, &db.CreateSessionParams{
		UserID:                     userID,
		SessionDeviceID:            deviceID,
		SessionUserAgent:           userAgent,
		SessionIp:                  ip,
		SessionProvider:            db.AuthAuthProviderApple,
		SessionRefreshTokenHmacKey: hmacKey,
		SessionNotAfter:            &sessionNotAfter,
	})
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		return nil, fmt.Errorf("create session: %w", err)
	}
	s.logger.Debug("session created", "session_id", pgToUUID(session.SessionID))
	s.logger.Debug("creating refresh token record")
	tokenHash := hashToken(hmacKey, session.SessionRefreshTokenCounter)
	_, err = qtx.CreateRefreshToken(ctx, &db.CreateRefreshTokenParams{
		SessionID:             session.SessionID,
		RefreshTokenTokenHash: tokenHash,
		RefreshTokenCounter:   session.SessionRefreshTokenCounter,
	})
	if err != nil {
		s.logger.Error("failed to create refresh token", "error", err)
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	s.logger.Debug("committing transaction")
	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("failed to commit transaction", "error", err)
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	sessionUUID := pgToUUID(session.SessionID)
	userUUID := pgToUUID(userID)
	appleTokenExpiry := time.Now().Add(180 * 24 * time.Hour)
	s.logger.Debug("generating token pair")
	tokens, err := s.generateTokenPair(ctx, sessionUUID, userUUID, session.SessionRefreshTokenCounter, sessionNotAfter, tokenResp.RefreshToken, appleTokenExpiry.Unix())
	if err != nil {
		s.logger.Error("failed to generate tokens", "error", err)
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	s.logger.Info("apple sign in completed successfully",
		"user_id", userUUID,
		"session_id", sessionUUID,
		"is_new_user", isNewUser,
	)
	return &SignInWithAppleResponse{
		Tokens:    tokens,
		UserID:    userUUID,
		IsNewUser: isNewUser,
	}, nil
}

type RefreshTokensRequest struct {
	RefreshToken string
}

type RefreshTokensResponse struct {
	Tokens TokenPair
	UserID uuid.UUID
}

func (s *Service) RefreshTokens(ctx context.Context, req RefreshTokensRequest) (*RefreshTokensResponse, error) {
	claims, err := s.jwtService.VerifyRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)
	session, err := s.getActiveSessionForUpdate(ctx, tx, claims.SessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if claims.Counter != session.SessionRefreshTokenCounter {
		s.logger.Warn("refresh token reuse detected",
			"session_id", session.SessionID,
			"expected_counter", session.SessionRefreshTokenCounter,
			"received_counter", claims.Counter,
		)
		if err := qtx.RevokeSession(ctx, session.SessionID); err != nil {
			return nil, fmt.Errorf("revoke session after reuse: %w", err)
		}
		if err := qtx.RevokeAllSessionRefreshTokens(ctx, session.SessionID); err != nil {
			return nil, fmt.Errorf("revoke refresh tokens after reuse: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit token reuse revocation: %w", err)
		}
		return nil, ErrTokenReuse
	}
	newAppleRefreshToken := claims.AppleRefreshToken
	newAppleRefreshExp := claims.AppleRefreshExp
	if s.appleClient != nil && session.SessionProvider == db.AuthAuthProviderApple && claims.AppleRefreshToken != "" {
		newAppleTokens, refreshErr := s.appleClient.RefreshAccessToken(ctx, claims.AppleRefreshToken)
		if refreshErr != nil {
			s.logger.Warn("failed to refresh apple token", "error", refreshErr)
		} else {
			if newAppleTokens.RefreshToken != "" {
				newAppleRefreshToken = newAppleTokens.RefreshToken
				newAppleRefreshExp = time.Now().Add(180 * 24 * time.Hour).Unix()
			}
		}
	}
	updatedSession, err := qtx.UpdateSessionRefreshed(ctx, session.SessionID)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}
	oldToken, err := qtx.GetRefreshTokenBySessionAndCounter(ctx, &db.GetRefreshTokenBySessionAndCounterParams{
		SessionID:           session.SessionID,
		RefreshTokenCounter: claims.Counter,
	})
	if err == nil {
		_ = qtx.RevokeRefreshToken(ctx, oldToken.RefreshTokenID)
	}
	tokenHash := hashToken(session.SessionRefreshTokenHmacKey, updatedSession.SessionRefreshTokenCounter)
	_, err = qtx.CreateRefreshToken(ctx, &db.CreateRefreshTokenParams{
		SessionID:             session.SessionID,
		RefreshTokenTokenHash: tokenHash,
		RefreshTokenCounter:   updatedSession.SessionRefreshTokenCounter,
	})
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	sessionNotAfter := time.Now().Add(RefreshTokenExpiry)
	if session.SessionNotAfter != nil {
		sessionNotAfter = *session.SessionNotAfter
	}
	sessionUUID := pgToUUID(session.SessionID)
	userUUID := pgToUUID(session.UserID)
	tokens, err := s.generateTokenPair(ctx, sessionUUID, userUUID, updatedSession.SessionRefreshTokenCounter, sessionNotAfter, newAppleRefreshToken, newAppleRefreshExp)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	return &RefreshTokensResponse{
		Tokens: tokens,
		UserID: userUUID,
	}, nil
}

func (s *Service) SignOut(ctx context.Context, sessionID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)
	pgSessionID := uuidToPg(sessionID)
	if err := qtx.RevokeSession(ctx, pgSessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if err := qtx.RevokeAllSessionRefreshTokens(ctx, pgSessionID); err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) SignOutWithOwnershipCheck(ctx context.Context, userID, sessionID uuid.UUID) error {
	pgSessionID := uuidToPg(sessionID)
	session, err := s.queries.GetSessionByID(ctx, pgSessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("get session: %w", err)
	}
	if pgToUUID(session.UserID) != userID {
		return ErrSessionNotOwned
	}
	return s.SignOut(ctx, sessionID)
}

func (s *Service) SignOutAllSessions(ctx context.Context, userID uuid.UUID) error {
	return s.queries.RevokeAllUserSessions(ctx, uuidToPg(userID))
}

func (s *Service) VerifyAccessToken(ctx context.Context, tokenString string) (*AccessTokenClaims, error) {
	claims, err := s.jwtService.VerifyAccessToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	_, err = s.queries.GetActiveSessionByID(ctx, uuidToPg(claims.SessionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionRevoked
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return claims, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (db.AuthUser, error) {
	user, err := s.queries.GetUserByID(ctx, uuidToPg(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AuthUser{}, ErrUserNotFound
		}
		return db.AuthUser{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *Service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]db.AuthSession, error) {
	return s.queries.GetSessionsByUserID(ctx, uuidToPg(userID))
}

func (s *Service) generateTokenPair(ctx context.Context, sessionID, userID uuid.UUID, counter int64, refreshExpiry time.Time, appleRefreshToken string, appleRefreshExp int64) (TokenPair, error) {
	pgUserID := uuidToPg(userID)
	roles, err := s.queries.GetUserRoles(ctx, pgUserID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("get user roles: %w", err)
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.RoleName
	}
	flags, err := s.queries.GetActiveFeatureFlags(ctx, pgUserID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("get feature flags: %w", err)
	}
	now := time.Now()
	accessExpiry := now.Add(AccessTokenExpiry)
	accessClaims := AccessTokenClaims{
		UserID:       userID,
		SessionID:    sessionID,
		Roles:        roleNames,
		FeatureFlags: flags,
		IssuedAt:     now,
		ExpiresAt:    accessExpiry,
	}
	accessToken, err := s.jwtService.SignAccessToken(accessClaims)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	refreshClaims := RefreshTokenClaims{
		SessionID:         sessionID,
		Counter:           counter,
		IssuedAt:          now,
		ExpiresAt:         refreshExpiry,
		AppleRefreshToken: appleRefreshToken,
		AppleRefreshExp:   appleRefreshExp,
	}
	refreshToken, err := s.jwtService.SignRefreshToken(refreshClaims)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign refresh token: %w", err)
	}
	return TokenPair{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiry.Unix(),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiry.Unix(),
	}, nil
}

func (s *Service) JWTService() *JWTService {
	return s.jwtService
}

func (s *Service) AppleClient() *apple.Client {
	return s.appleClient
}

func generateHMACKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func hashToken(hmacKey string, counter int64) string {
	data := fmt.Sprintf("%s:%d", hmacKey, counter)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *Service) getActiveSessionForUpdate(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) (db.AuthSession, error) {
	const query = `
SELECT session_id, user_id, session_device_id, session_user_agent, session_ip, session_provider,
       session_refresh_token_hmac_key, session_refresh_token_counter, session_created_at, session_updated_at,
       session_refreshed_at, session_not_after, session_revoked_at
FROM auth.sessions
WHERE session_id = $1
  AND session_revoked_at IS NULL
  AND (session_not_after IS NULL OR session_not_after > now())
FOR UPDATE`
	row := tx.QueryRow(ctx, query, sessionID)
	var session db.AuthSession
	err := row.Scan(
		&session.SessionID,
		&session.UserID,
		&session.SessionDeviceID,
		&session.SessionUserAgent,
		&session.SessionIp,
		&session.SessionProvider,
		&session.SessionRefreshTokenHmacKey,
		&session.SessionRefreshTokenCounter,
		&session.SessionCreatedAt,
		&session.SessionUpdatedAt,
		&session.SessionRefreshedAt,
		&session.SessionNotAfter,
		&session.SessionRevokedAt,
	)
	return session, err
}

func pgToUUID(pg pgtype.UUID) uuid.UUID {
	if !pg.Valid {
		return uuid.Nil
	}
	return uuid.UUID(pg.Bytes)
}

func uuidToPg(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}
