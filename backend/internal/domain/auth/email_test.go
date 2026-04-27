package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	db "koditon-go/internal/db"
	"koditon-go/internal/domain/emailauth"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSignInWithEmail_ReusesIdentityAcrossConcurrentRequests(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL (or LOCAL_DATABASE_URL) to run DB-backed auth test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	emailAuthService := emailauth.NewService(emailauth.ServiceConfig{
		Logger:  slog.Default(),
		Queries: queries,
	})
	authService := &Service{
		logger:  slog.Default(),
		pool:    pool,
		queries: queries,
	}

	rawEmail := "Race.User+test@example.com"
	authenticatedEmail := createAuthenticatedEmailForTest(t, ctx, queries, emailAuthService, rawEmail)
	normalizedEmail := authenticatedEmail.String()

	t.Cleanup(func() {
		_, cleanupErr := pool.Exec(ctx, "delete from users where lower(btrim(user_email)) = lower(btrim($1))", normalizedEmail)
		if cleanupErr != nil {
			t.Fatalf("cleanup user: %v", cleanupErr)
		}
		_, cleanupErr = pool.Exec(ctx, "delete from auth_signup_email_tokens where lower(btrim(auth_signup_email_target_email)) = lower(btrim($1))", normalizedEmail)
		if cleanupErr != nil {
			t.Fatalf("cleanup signup email tokens: %v", cleanupErr)
		}
		_, cleanupErr = pool.Exec(ctx, "delete from auth_signup_tickets where lower(btrim(auth_signup_ticket_target_email)) = lower(btrim($1))", normalizedEmail)
		if cleanupErr != nil {
			t.Fatalf("cleanup signup tickets: %v", cleanupErr)
		}
	})

	results := make(chan *SignInWithEmailResponse, 2)
	errs := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	runSignIn := func() {
		defer wg.Done()
		<-start
		resp, signInErr := authService.SignInWithEmail(ctx, SignInWithEmailRequest{
			ConfirmedEmail: authenticatedEmail,
		})
		if signInErr != nil {
			errs <- signInErr
			return
		}
		results <- resp
	}

	go runSignIn()
	go runSignIn()
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("sign in failed: %v", err)
		}
	}

	var responses []*SignInWithEmailResponse
	for resp := range results {
		responses = append(responses, resp)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[0].UserID != responses[1].UserID {
		t.Fatalf("expected same user id, got %s and %s", responses[0].UserID, responses[1].UserID)
	}
	if responses[0].SessionID == responses[1].SessionID {
		t.Fatalf("expected distinct sessions, got duplicate %s", responses[0].SessionID)
	}

	const countEmailIdentitiesSQL = `
select count(*)
from user_identities
where user_identity_provider = 'email'
  and user_identity_external_id = $1
`
	var identityCount int
	if err := pool.QueryRow(ctx, countEmailIdentitiesSQL, normalizedEmail).Scan(&identityCount); err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if identityCount != 1 {
		t.Fatalf("expected exactly 1 email identity, got %d", identityCount)
	}
}

func TestSignInWithEmail_RepairsMissingEmailIdentityFromLegacyUserEmail(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL (or LOCAL_DATABASE_URL) to run DB-backed auth test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	emailAuthService := emailauth.NewService(emailauth.ServiceConfig{
		Logger:  slog.Default(),
		Queries: queries,
	})
	authService := &Service{
		logger:  slog.Default(),
		pool:    pool,
		queries: queries,
	}

	rawEmail := "Legacy.User+repair@example.com"
	normalizedEmail, err := emailauth.NormalizeEmail(rawEmail)
	if err != nil {
		t.Fatalf("normalize email: %v", err)
	}

	user, err := queries.CreateUser(ctx, &normalizedEmail)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, cleanupErr := pool.Exec(ctx, "delete from users where user_uuid = $1", user.UserUuid)
		if cleanupErr != nil {
			t.Fatalf("cleanup user: %v", cleanupErr)
		}
		_, cleanupErr = pool.Exec(ctx, "delete from auth_signup_email_tokens where lower(btrim(auth_signup_email_target_email)) = lower(btrim($1))", normalizedEmail)
		if cleanupErr != nil {
			t.Fatalf("cleanup signup email tokens: %v", cleanupErr)
		}
		_, cleanupErr = pool.Exec(ctx, "delete from auth_signup_tickets where lower(btrim(auth_signup_ticket_target_email)) = lower(btrim($1))", normalizedEmail)
		if cleanupErr != nil {
			t.Fatalf("cleanup signup tickets: %v", cleanupErr)
		}
	})

	authenticatedEmail := createAuthenticatedEmailForTest(t, ctx, queries, emailAuthService, rawEmail)
	resp, err := authService.SignInWithEmail(ctx, SignInWithEmailRequest{
		ConfirmedEmail: authenticatedEmail,
	})
	if err != nil {
		t.Fatalf("sign in with email: %v", err)
	}
	if resp.UserID != user.UserUuid {
		t.Fatalf("expected repaired sign-in to reuse user %s, got %s", user.UserUuid, resp.UserID)
	}

	emailProvider := string(AuthProviderEmail)
	identity, err := queries.GetIdentityByProviderAndExternalID(ctx, db.GetIdentityByProviderAndExternalIDParams{
		UserIdentityProvider:   emailProvider,
		UserIdentityExternalID: normalizedEmail,
	})
	if err != nil {
		t.Fatalf("get repaired email identity: %v", err)
	}
	if identity.UserUuid != user.UserUuid {
		t.Fatalf("expected repaired identity to belong to user %s, got %s", user.UserUuid, identity.UserUuid)
	}
}

func createAuthenticatedEmailForTest(t *testing.T, ctx context.Context, queries *db.Queries, service *emailauth.Service, rawEmail string) emailauth.AuthenticatedEmail {
	t.Helper()

	normalizedEmail, err := emailauth.NormalizeEmail(rawEmail)
	if err != nil {
		t.Fatalf("normalize email: %v", err)
	}

	rawToken := "signup-email-token-" + time.Now().UTC().Format(time.RFC3339Nano)
	tokenHash := sha256.Sum256([]byte(rawToken))
	tokenHashHex := hex.EncodeToString(tokenHash[:])
	if _, err := queries.CreateSignupEmailToken(ctx, db.CreateSignupEmailTokenParams{
		AuthSignupEmailTargetEmail: normalizedEmail,
		AuthSignupEmailTokenHash:   tokenHashHex,
		AuthSignupEmailExpiresAt:   time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create signup email token: %v", err)
	}

	rawTicket, err := service.ConfirmAuthentication(ctx, rawToken)
	if err != nil {
		t.Fatalf("confirm authentication: %v", err)
	}
	authenticatedEmail, err := service.ConsumeAuthenticationTicket(ctx, rawTicket)
	if err != nil {
		t.Fatalf("consume authentication ticket: %v", err)
	}
	return authenticatedEmail
}
