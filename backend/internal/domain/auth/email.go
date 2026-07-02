package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	db "koditon/internal/db"
	"koditon/internal/domain/emailauth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EmailAuthenticationMethod string

const (
	EmailAuthenticationMethodPasskey   EmailAuthenticationMethod = "passkey"
	EmailAuthenticationMethodMagicLink EmailAuthenticationMethod = "magic_link"
)

type BeginEmailAuthenticationResponse struct {
	Method      EmailAuthenticationMethod
	ChallengeID uuid.UUID
	Options     json.RawMessage
}

type SignInWithEmailRequest struct {
	ConfirmedEmail   emailauth.AuthenticatedEmail
	DeviceID         uuid.UUID
	DeviceName       string
	DeviceOS         string
	DeviceModel      string
	DeviceLocale     string
	DeviceTimeZone   string
	DeviceAppVersion string
	UserAgent        string
	IP               string
}

type SignInWithEmailResponse struct {
	UserID    uuid.UUID
	IsNewUser bool
	SessionID uuid.UUID
}

type resolvedEmailAccount struct {
	userID               uuid.UUID
	userIDBigint         int64
	identity             *db.GetIdentityByProviderAndExternalIDRow
	usedLegacyUserLookup bool
}

func (s *Service) BeginEmailAuthentication(ctx context.Context, rawEmail string) (*BeginEmailAuthenticationResponse, error) {
	if _, err := emailauth.NormalizeEmail(rawEmail); err != nil {
		return nil, err
	}
	return &BeginEmailAuthenticationResponse{
		Method: EmailAuthenticationMethodMagicLink,
	}, nil
}

func (s *Service) SignInWithEmail(ctx context.Context, req SignInWithEmailRequest) (*SignInWithEmailResponse, error) {
	normalizedEmail := req.ConfirmedEmail.String()
	if req.ConfirmedEmail.IsZero() || normalizedEmail == "" {
		return nil, ErrOAuthInvalidRequest
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(ctx, s.logger, tx)
	qtx := s.queries.WithTx(tx)

	emailProvider := AuthProviderEmail
	identityData, _ := json.Marshal(map[string]any{
		"email": normalizedEmail,
	})

	var userID uuid.UUID
	var userIDBigint int64
	isNewUser := false
	account, err := s.resolveEmailAccount(ctx, qtx, normalizedEmail, true)
	if err != nil {
		return nil, err
	}

	switch {
	case account != nil && account.identity != nil:
		userID = account.userID
		userIDBigint = account.userIDBigint
		_, err = qtx.UpdateIdentity(ctx, db.UpdateIdentityParams{
			UserIdentityUuid:          new(account.identity.UserIdentityUuid),
			UserIdentityEmail:         &normalizedEmail,
			UserIdentityEmailVerified: new(true),
			UserIdentityData:          identityData,
		})
		if err != nil {
			return nil, fmt.Errorf("update email identity: %w", err)
		}
		if _, err := qtx.UpdateUserEmailIfEmptyByIDBigint(ctx, db.UpdateUserEmailIfEmptyByIDBigintParams{
			UserEmail: &normalizedEmail,
			UserID:    new(userIDBigint),
		}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update user email if empty: %w", err)
		}
	case account != nil && account.usedLegacyUserLookup:
		s.logger.WarnContext(ctx, "repairing missing email identity from legacy user_email lookup", "user_id", account.userID, "email", normalizedEmail)
		userID = account.userID
		if _, err := qtx.CreateIdentity(ctx, db.CreateIdentityParams{
			UserUuid:                  new(userID),
			UserIdentityProvider:      new(string(emailProvider)),
			UserIdentityExternalID:    new(normalizedEmail),
			UserIdentityEmail:         &normalizedEmail,
			UserIdentityEmailVerified: new(true),
			UserIdentityData:          identityData,
		}); err != nil {
			if !isUniqueViolation(err) {
				return nil, fmt.Errorf("repair email identity: %w", err)
			}
			account, err = s.resolveEmailAccount(ctx, qtx, normalizedEmail, false)
			if err != nil {
				return nil, err
			}
			if account == nil || account.identity == nil {
				return nil, fmt.Errorf("reload repaired email identity: %w", pgx.ErrNoRows)
			}
			userID = account.userID
		}
	default:
		userID, userIDBigint, isNewUser, err = s.findOrCreateUserByEmail(ctx, qtx, normalizedEmail)
		if err != nil {
			return nil, err
		}

		if !isNewUser {
			if _, err := qtx.UpdateUserEmailIfEmptyByIDBigint(ctx, db.UpdateUserEmailIfEmptyByIDBigintParams{
				UserEmail: &normalizedEmail,
				UserID:    new(userIDBigint),
			}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("update user email if empty: %w", err)
			}
		}

		if _, err := qtx.CreateIdentity(ctx, db.CreateIdentityParams{
			UserUuid:                  new(userID),
			UserIdentityProvider:      new(string(emailProvider)),
			UserIdentityExternalID:    new(normalizedEmail),
			UserIdentityEmail:         &normalizedEmail,
			UserIdentityEmailVerified: new(true),
			UserIdentityData:          identityData,
		}); err != nil {
			if !isUniqueViolation(err) {
				return nil, fmt.Errorf("create email identity: %w", err)
			}
			account, err = s.resolveEmailAccount(ctx, qtx, normalizedEmail, false)
			if err != nil {
				return nil, fmt.Errorf("reload email identity after unique violation: %w", err)
			}
			if account == nil || account.identity == nil {
				return nil, fmt.Errorf("reload email identity after unique violation: %w", pgx.ErrNoRows)
			}
			userID = account.userID
			isNewUser = false
		}
	}

	sessionID, err := s.createSessionWithProvider(ctx, tx, createSessionParams{
		UserID:           userID,
		Provider:         AuthProviderEmail,
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
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &SignInWithEmailResponse{
		UserID:    userID,
		IsNewUser: isNewUser,
		SessionID: sessionID,
	}, nil
}

func (s *Service) resolveEmailAccount(
	ctx context.Context,
	queries *db.Queries,
	normalizedEmail string,
	allowLegacyUserLookup bool,
) (*resolvedEmailAccount, error) {
	emailProvider := AuthProviderEmail
	identity, err := queries.GetIdentityByProviderAndExternalID(ctx, db.GetIdentityByProviderAndExternalIDParams{
		UserIdentityProvider:   new(string(emailProvider)),
		UserIdentityExternalID: new(normalizedEmail),
	})
	switch {
	case err == nil:
		return &resolvedEmailAccount{
			userID:       *identity.UserUuid,
			userIDBigint: identity.UserIDBigint,
			identity:     &identity,
		}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("get email identity: %w", err)
	case !allowLegacyUserLookup:
		return nil, nil
	}

	userRow, err := queries.GetUserByEmail(ctx, &normalizedEmail)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("get user by email: %w", err)
	default:
		return &resolvedEmailAccount{
			userID:               userRow.UserUuid,
			userIDBigint:         userRow.UserIDBigint,
			usedLegacyUserLookup: true,
		}, nil
	}
}

func (s *Service) findOrCreateUserByEmail(ctx context.Context, qtx *db.Queries, normalizedEmail string) (uuid.UUID, int64, bool, error) {
	userRow, lookupErr := qtx.GetUserByEmail(ctx, &normalizedEmail)
	switch {
	case lookupErr == nil:
		return userRow.UserUuid, userRow.UserIDBigint, false, nil
	case !errors.Is(lookupErr, pgx.ErrNoRows):
		return uuid.Nil, 0, false, fmt.Errorf("get user by email: %w", lookupErr)
	}

	newUser, createErr := qtx.CreateUser(ctx, &normalizedEmail)
	if createErr != nil {
		if !isUniqueViolation(createErr) {
			return uuid.Nil, 0, false, fmt.Errorf("create user: %w", createErr)
		}
		userRow, lookupErr = qtx.GetUserByEmail(ctx, &normalizedEmail)
		if lookupErr != nil {
			return uuid.Nil, 0, false, fmt.Errorf("reload user after unique violation: %w", lookupErr)
		}
		return userRow.UserUuid, userRow.UserIDBigint, false, nil
	}
	return newUser.UserUuid, newUser.UserIDBigint, true, nil
}
