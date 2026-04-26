package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"koditon-go/internal/auth/apple"
	"koditon-go/internal/auth/passkey"
	db "koditon-go/internal/db"
	"koditon-go/internal/logging"
	"koditon-go/internal/runtimecfg"
	"koditon-go/internal/util"

	"github.com/go-webauthn/webauthn/protocol"
	wbauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	logger              *slog.Logger
	pool                *pgxpool.Pool
	queries             *db.Queries
	jwtService          *JWTService
	appleClient         *apple.Client
	appleWebClient      *apple.Client
	appleWebRedirectURI string
	passkeyService      passkeyCeremonyService
	geoResolver         GeoResolver
	policy              Policy
}

type passkeyCeremonyService interface {
	BeginDiscoverableAuthentication() (*protocol.CredentialAssertion, *wbauthn.SessionData, error)
	BeginAuthentication(user passkey.User) (*protocol.CredentialAssertion, *wbauthn.SessionData, error)
	BeginRegistration(user passkey.User, exclude []protocol.CredentialDescriptor) (*protocol.CredentialCreation, *wbauthn.SessionData, error)
	FinishRegistration(ctx context.Context, user passkey.User, session wbauthn.SessionData, credentialJSON []byte) (*wbauthn.Credential, error)
	FinishPasskeyLogin(ctx context.Context, session wbauthn.SessionData, credentialJSON []byte, handler wbauthn.DiscoverableUserHandler) (wbauthn.User, *wbauthn.Credential, error)
}

type ServiceConfig struct {
	Pool        *pgxpool.Pool
	Auth        *runtimecfg.AuthConfig
	RedisClient *redis.Client
	GeoResolver GeoResolver
	Logger      *slog.Logger
	Policy      PolicyConfig
}

func NewService(ctx context.Context, cfg ServiceConfig) (*Service, error) {
	if cfg.Auth == nil {
		return nil, errors.New("auth config is required")
	}
	jwtService, err := NewJWTService(JWTConfig{
		PrivateKey:  cfg.Auth.JWT.PrivateKey,
		Issuer:      cfg.Auth.JWT.Issuer,
		UIDHashSalt: cfg.Auth.JWT.UIDHashSalt,
	})
	if err != nil {
		return nil, fmt.Errorf("create jwt service: %w", err)
	}
	appleClient, err := apple.NewClient(ctx, apple.Config{
		BundleID:     cfg.Auth.Apple.BundleID,
		TeamID:       cfg.Auth.Apple.TeamID,
		PrivateKeyID: cfg.Auth.Apple.PrivateKeyID,
		PrivateKey:   cfg.Auth.Apple.PrivateKey,
		RedisClient:  cfg.RedisClient,
	})
	if err != nil {
		return nil, fmt.Errorf("create apple client: %w", err)
	}
	var appleWebClient *apple.Client
	if cfg.Auth.Apple.WebServiceID != "" {
		appleWebClient, err = apple.NewClient(ctx, apple.Config{
			BundleID:     cfg.Auth.Apple.WebServiceID,
			TeamID:       cfg.Auth.Apple.TeamID,
			PrivateKeyID: cfg.Auth.Apple.PrivateKeyID,
			PrivateKey:   cfg.Auth.Apple.PrivateKey,
			RedisClient:  cfg.RedisClient,
		})
		if err != nil {
			return nil, fmt.Errorf("create apple web client: %w", err)
		}
	}
	var passkeyService *passkey.Service
	if cfg.Auth.Passkey != nil {
		passkeyService, err = passkey.NewService(passkey.Config{
			RPID:          cfg.Auth.Passkey.RPID,
			RPDisplayName: cfg.Auth.Passkey.RPDisplayName,
			RPOrigins:     cfg.Auth.Passkey.RPOrigins,
		})
		if err != nil {
			return nil, fmt.Errorf("create passkey service: %w", err)
		}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	geoResolver := cfg.GeoResolver
	if geoResolver == nil {
		geoResolver = noopGeoResolver{}
	}
	return &Service{
		logger:              logger.With("component", "auth"),
		pool:                cfg.Pool,
		queries:             db.New(cfg.Pool),
		jwtService:          jwtService,
		appleClient:         appleClient,
		appleWebClient:      appleWebClient,
		appleWebRedirectURI: cfg.Auth.Apple.WebRedirectURI,
		passkeyService:      passkeyService,
		geoResolver:         geoResolver,
		policy:              newPolicy(cfg.Policy),
	}, nil
}

func (s *Service) Close() error {
	return closeGeoResolver(s.geoResolver)
}

type SignInWithAppleRequest struct {
	AuthorizationCode string
	Nonce             string
	DeviceID          uuid.UUID
	DeviceName        string
	DeviceOS          string
	DeviceModel       string
	DeviceLocale      string
	DeviceTimeZone    string
	DeviceAppVersion  string
	UserAgent         string
	IP                string
}

type SignInWithAppleResponse struct {
	UserID    uuid.UUID
	IsNewUser bool
	SessionID uuid.UUID
}

func rollbackTx(ctx context.Context, logger *slog.Logger, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		logger.ErrorContext(ctx, "rollback transaction failed", "error", err)
	}
}

func (s *Service) SignInWithApple(ctx context.Context, req SignInWithAppleRequest) (*SignInWithAppleResponse, error) {
	logger := logging.With(s.logger, logging.Op("auth.apple_sign_in"))
	logger.DebugContext(ctx, "apple sign in started",
		"has_auth_code", req.AuthorizationCode != "",
		"has_nonce", req.Nonce != "",
		"has_device_id", req.DeviceID != uuid.Nil,
	)
	if s.appleClient == nil {
		logger.ErrorContext(ctx, "apple client not configured", "outcome", logging.OutcomeError)
		return nil, errors.New("apple client not configured")
	}
	logger.DebugContext(ctx, "exchanging authorization code with apple")
	tokenResp, err := s.appleClient.ExchangeAuthorizationCode(ctx, req.AuthorizationCode)
	if err != nil {
		var exchangeErr *apple.TokenExchangeError
		if errors.As(err, &exchangeErr) {
			logger.ErrorContext(ctx, "failed to exchange authorization code",
				"error", err,
				"apple_status_code", exchangeErr.StatusCode,
				"apple_error_code", exchangeErr.ErrorCode,
				"apple_error_description", exchangeErr.ErrorDescription,
				"apple_bundle_id", s.appleClient.BundleID(),
			)
		} else {
			logger.ErrorContext(ctx, "failed to exchange authorization code", "error", err, "outcome", logging.OutcomeError)
		}
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	logger.DebugContext(ctx, "authorization code exchanged successfully",
		"has_id_token", tokenResp.IDToken != "",
		"has_access_token", tokenResp.AccessToken != "",
		"has_refresh_token", tokenResp.RefreshToken != "",
	)
	if tokenResp.IDToken == "" {
		logger.ErrorContext(ctx, "apple returned empty id token", "outcome", logging.OutcomeError)
		return nil, ErrMissingIDToken
	}
	logger.DebugContext(ctx, "verifying apple id token")
	identity, err := s.appleClient.VerifyIDToken(ctx, tokenResp.IDToken, req.Nonce)
	if err != nil {
		logger.ErrorContext(ctx, "failed to verify id token", "error", err, "outcome", logging.OutcomeError)
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	logger.DebugContext(ctx, "id token verified",
		"email_verified", identity.EmailVerified,
	)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
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
	logger.DebugContext(ctx, "looking up existing identity", "provider", "apple", "external_id", identity.Subject)
	provider := AuthProviderApple
	existingIdentity, err := qtx.GetIdentityByProviderAndExternalID(ctx, db.GetIdentityByProviderAndExternalIDParams{
		UserIdentityProvider:   &provider,
		UserIdentityExternalID: &identity.Subject,
	})
	var userID uuid.UUID
	var isNewUser bool
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			logger.ErrorContext(ctx, "failed to lookup identity", "error", err, "outcome", logging.OutcomeError)
			return nil, fmt.Errorf("get identity: %w", err)
		}
		logger.DebugContext(ctx, "no existing identity found, creating new user")
		user, createErr := qtx.CreateUser(ctx, email)
		if createErr != nil {
			logger.ErrorContext(ctx, "failed to create user", "error", createErr, "outcome", logging.OutcomeError)
			return nil, fmt.Errorf("create user: %w", createErr)
		}
		userID = user.UserUuid
		isNewUser = true
		logger.DebugContext(ctx, "creating identity for new user", "user_id", userID)
		appleProvider := AuthProviderApple
		_, err = qtx.CreateIdentity(ctx, db.CreateIdentityParams{
			UserUuid:                  util.UUIDToPg(userID),
			UserIdentityProvider:      &appleProvider,
			UserIdentityExternalID:    &identity.Subject,
			UserIdentityEmail:         email,
			UserIdentityEmailVerified: &emailVerified,
			UserIdentityData:          identityData,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to create identity", "error", err, "user_id", userID, "outcome", logging.OutcomeError)
			return nil, fmt.Errorf("create identity: %w", err)
		}
		logger.InfoContext(ctx, "new user created via apple sign in", "user_id", userID, "outcome", logging.OutcomeSuccess)
	} else {
		userID = util.PgUUIDToUUID(existingIdentity.UserUuid)
		logger.DebugContext(ctx, "existing user found", "user_id", userID, "identity_id", existingIdentity.UserIdentityUuid)
		_, err = qtx.UpdateIdentity(ctx, db.UpdateIdentityParams{
			UserIdentityUuid:          util.UUIDToPg(existingIdentity.UserIdentityUuid),
			UserIdentityEmail:         email,
			UserIdentityEmailVerified: &emailVerified,
			UserIdentityData:          identityData,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to update identity", "error", err, "user_id", userID, "outcome", logging.OutcomeError)
			return nil, fmt.Errorf("update identity: %w", err)
		}
		if email != nil {
			_, err = qtx.UpdateUserEmailIfEmptyByIDBigint(ctx, db.UpdateUserEmailIfEmptyByIDBigintParams{
				UserEmail: email,
				UserID:    &existingIdentity.UserIDBigint,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				logger.ErrorContext(ctx, "failed to update user email if empty", "error", err, "user_id", userID, "outcome", logging.OutcomeError)
				return nil, fmt.Errorf("update user email if empty: %w", err)
			}
		}
		logger.DebugContext(ctx, "identity updated successfully", "user_id", userID)
	}
	logger.DebugContext(ctx, "creating session", "user_id", userID, "has_device_id", req.DeviceID != uuid.Nil)
	sessionID, err := s.createSessionWithProvider(ctx, tx, createSessionParams{
		UserID:           userID,
		Provider:         AuthProviderApple,
		DeviceID:         req.DeviceID,
		DeviceName:       req.DeviceName,
		DeviceOS:         req.DeviceOS,
		DeviceModel:      req.DeviceModel,
		DeviceLocale:     req.DeviceLocale,
		DeviceTimeZone:   req.DeviceTimeZone,
		DeviceAppVersion: req.DeviceAppVersion,
		UserAgent:        req.UserAgent,
		IP:               req.IP,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create session", "error", err, "user_id", userID, "outcome", logging.OutcomeError)
		return nil, err
	}
	logger.DebugContext(ctx, "session created", "session_id", sessionID)
	logger.DebugContext(ctx, "committing transaction")
	if err := tx.Commit(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to commit transaction", "error", err, "user_id", userID, "session_id", sessionID, "outcome", logging.OutcomeError)
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	logger.InfoContext(ctx, "apple sign in completed successfully",
		"user_id", userID,
		"session_id", sessionID,
		"is_new_user", isNewUser,
		"outcome", logging.OutcomeSuccess,
	)
	return &SignInWithAppleResponse{
		UserID:    userID,
		IsNewUser: isNewUser,
		SessionID: sessionID,
	}, nil
}

type SignInWithAppleWebRequest struct {
	AuthorizationCode string
	DeviceID          uuid.UUID
	UserAgent         string
	IP                string
}

func (s *Service) SignInWithAppleWeb(ctx context.Context, req SignInWithAppleWebRequest) (*SignInWithAppleResponse, error) {
	logger := logging.With(s.logger, logging.Op("auth.apple_web_sign_in"))
	if s.appleWebClient == nil {
		return nil, errors.New("apple web sign in not configured")
	}
	tokenResp, err := s.appleWebClient.ExchangeAuthorizationCodeWeb(ctx, req.AuthorizationCode, s.appleWebRedirectURI)
	if err != nil {
		logger.ErrorContext(ctx, "failed to exchange web authorization code", "error", err, "outcome", logging.OutcomeError)
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	if tokenResp.IDToken == "" {
		return nil, ErrMissingIDToken
	}
	identity, err := s.appleWebClient.VerifyIDToken(ctx, tokenResp.IDToken, "")
	if err != nil {
		logger.ErrorContext(ctx, "failed to verify web id token", "error", err, "outcome", logging.OutcomeError)
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
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
		"email_verified": emailVerified,
		"phone_verified": false,
	})
	provider := AuthProviderApple
	existingIdentity, err := qtx.GetIdentityByProviderAndExternalID(ctx, db.GetIdentityByProviderAndExternalIDParams{
		UserIdentityProvider:   &provider,
		UserIdentityExternalID: &identity.Subject,
	})
	var userID uuid.UUID
	var isNewUser bool
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get identity: %w", err)
		}
		user, createErr := qtx.CreateUser(ctx, email)
		if createErr != nil {
			return nil, fmt.Errorf("create user: %w", createErr)
		}
		userID = user.UserUuid
		isNewUser = true
		appleProvider := AuthProviderApple
		_, err = qtx.CreateIdentity(ctx, db.CreateIdentityParams{
			UserUuid:                  util.UUIDToPg(userID),
			UserIdentityProvider:      &appleProvider,
			UserIdentityExternalID:    &identity.Subject,
			UserIdentityEmail:         email,
			UserIdentityEmailVerified: &emailVerified,
			UserIdentityData:          identityData,
		})
		if err != nil {
			return nil, fmt.Errorf("create identity: %w", err)
		}
	} else {
		userID = util.PgUUIDToUUID(existingIdentity.UserUuid)
		_, err = qtx.UpdateIdentity(ctx, db.UpdateIdentityParams{
			UserIdentityUuid:          util.UUIDToPg(existingIdentity.UserIdentityUuid),
			UserIdentityEmail:         email,
			UserIdentityEmailVerified: &emailVerified,
			UserIdentityData:          identityData,
		})
		if err != nil {
			return nil, fmt.Errorf("update identity: %w", err)
		}
	}
	sessionID, err := s.createSessionWithProvider(ctx, tx, createSessionParams{
		UserID:    userID,
		Provider:  AuthProviderApple,
		DeviceID:  req.DeviceID,
		UserAgent: req.UserAgent,
		IP:        req.IP,
	})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	logger.InfoContext(ctx, "apple web sign in completed", "user_id", userID, "session_id", sessionID, "is_new_user", isNewUser, "outcome", logging.OutcomeSuccess)
	return &SignInWithAppleResponse{
		UserID:    userID,
		IsNewUser: isNewUser,
		SessionID: sessionID,
	}, nil
}

func (s *Service) SignOut(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.queries.RevokeSession(ctx, util.UUIDToPg(sessionID)); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRevoked,
		AuthType:  "app",
		SessionID: sessionID,
		TokenType: "session",
	})
	return nil
}

func (s *Service) SignOutWithOwnershipCheck(ctx context.Context, userID, sessionID uuid.UUID) error {
	session, err := s.queries.GetSessionByID(ctx, util.UUIDToPg(sessionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("get session: %w", err)
	}
	if util.PgUUIDToUUID(session.UserUuid) != userID {
		return ErrSessionNotOwned
	}
	return s.SignOut(ctx, sessionID)
}

func (s *Service) SignOutAllSessions(ctx context.Context, userID uuid.UUID) error {
	if err := s.queries.RevokeAllUserSessions(ctx, util.UUIDToPg(userID)); err != nil {
		return err
	}
	if err := s.queries.RevokeAllOAuthRefreshTokensByUserID(ctx, util.UUIDToPg(userID)); err != nil {
		return fmt.Errorf("revoke all oauth refresh tokens: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRevoked,
		AuthType:  "app",
		UserID:    userID,
		TokenType: "refresh",
	})
	return nil
}

// RevokeOAuthRefreshToken revokes a specific OAuth refresh token (RFC 7009).
// Returns nil even if the token is already revoked or doesn't exist (per RFC 7009).
func (s *Service) RevokeOAuthRefreshToken(ctx context.Context, token string) error {
	tokenHash := hashSHA256Hex(token)
	row, err := s.queries.RevokeOAuthRefreshTokenByHash(ctx, &tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// RFC 7009: the server responds with HTTP 200 even if the token
			// was already invalid or unknown.
			return nil
		}
		return fmt.Errorf("revoke oauth refresh token: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRevoked,
		AuthType:  string(AccessTokenKindOAuth),
		ClientID:  row.OauthClientID,
		UserID:    row.UserUuid,
		Scopes:    row.OauthRefreshTokenScopes,
		TokenType: "refresh",
	})
	return nil
}

// RevokeOAuthRefreshTokenForClient revokes a specific OAuth refresh token for the given client.
// Returns nil even if the token is already revoked, unknown, or belongs to a different client.
func (s *Service) RevokeOAuthRefreshTokenForClient(ctx context.Context, clientID, token string) error {
	clientID = strings.TrimSpace(clientID)
	token = strings.TrimSpace(token)
	if clientID == "" || token == "" {
		return nil
	}
	row, err := s.queries.RevokeOAuthRefreshTokenByHashAndClientID(ctx, db.RevokeOAuthRefreshTokenByHashAndClientIDParams{
		OauthRefreshTokenTokenHash: new(hashSHA256Hex(token)),
		OauthClientID:              new(clientID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("revoke oauth refresh token for client: %w", err)
	}
	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRevoked,
		AuthType:  string(AccessTokenKindOAuth),
		ClientID:  row.OauthClientID,
		UserID:    row.UserUuid,
		Scopes:    row.OauthRefreshTokenScopes,
		TokenType: "refresh",
	})
	return nil
}

func (s *Service) VerifyAccessToken(ctx context.Context, tokenString string) (*AccessTokenClaims, error) {
	claims, err := s.jwtService.VerifyAccessToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	if claims.SessionID == uuid.Nil {
		// OAuth-issued access tokens are not bound to a device session row.
		return claims, nil
	}
	_, err = s.queries.GetActiveSessionByID(ctx, util.UUIDToPg(claims.SessionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionRevoked
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return claims, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (db.GetUserByIDRow, error) {
	user, err := s.queries.GetUserByID(ctx, util.UUIDToPg(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetUserByIDRow{}, ErrUserNotFound
		}
		return db.GetUserByIDRow{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *Service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]db.GetSessionsByUserIDRow, error) {
	return s.queries.GetSessionsByUserID(ctx, util.UUIDToPg(userID))
}

func (s *Service) GetRoleNamesByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := s.queries.GetUserRoles(ctx, util.UUIDToPg(userID))
	if err != nil {
		return nil, err
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.RoleName
	}
	return roleNames, nil
}

func (s *Service) GetActiveFeatureFlagsByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.queries.GetActiveFeatureFlags(ctx, util.UUIDToPg(userID))
}

func (s *Service) JWTService() *JWTService {
	return s.jwtService
}

func (s *Service) AppleClient() *apple.Client {
	return s.appleClient
}

func (s *Service) Logger() *slog.Logger {
	return s.logger
}
