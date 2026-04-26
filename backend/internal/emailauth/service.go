package emailauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	db "koditon-go/internal/db"
	"koditon-go/internal/email"
	"koditon-go/internal/emailaddr"
	"koditon-go/internal/runtimecfg"

	"github.com/jackc/pgx/v5"
)

const (
	defaultEmailTokenTTL = 24 * time.Hour
	defaultTicketTTL     = 10 * time.Minute
)

type ServiceConfig struct {
	Logger          *slog.Logger
	Queries         *db.Queries
	EmailService    *email.Service
	HTTP            runtimecfg.HTTPConfig
	EmailTokenTTL   time.Duration
	TicketTTL       time.Duration
	EmitConsoleLink bool
}

type Service struct {
	logger          *slog.Logger
	queries         *db.Queries
	emailService    *email.Service
	webBaseURL      string
	publicAPIURL    string
	emailTokenTTL   time.Duration
	ticketTTL       time.Duration
	emitConsoleLink bool
}

type AuthenticatedEmail struct {
	address emailaddr.Address
}

func NewService(cfg ServiceConfig) *Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	emailTokenTTL := cfg.EmailTokenTTL
	if emailTokenTTL <= 0 {
		emailTokenTTL = defaultEmailTokenTTL
	}
	ticketTTL := cfg.TicketTTL
	if ticketTTL <= 0 {
		ticketTTL = defaultTicketTTL
	}
	return &Service{
		logger:          logger.With("component", "email_auth"),
		queries:         cfg.Queries,
		emailService:    cfg.EmailService,
		webBaseURL:      strings.TrimSpace(cfg.HTTP.WebBaseURL),
		publicAPIURL:    strings.TrimSpace(cfg.HTTP.APIPublicBaseURL),
		emailTokenTTL:   emailTokenTTL,
		ticketTTL:       ticketTTL,
		emitConsoleLink: cfg.EmitConsoleLink,
	}
}

func NormalizeEmail(raw string) (string, error) {
	address, err := emailaddr.Parse(raw)
	if err != nil {
		return "", ErrInvalidEmail
	}
	return address.String(), nil
}

func (e AuthenticatedEmail) String() string {
	return e.address.String()
}

func (e AuthenticatedEmail) IsZero() bool {
	return e.address == ""
}

func (s *Service) RequestAuthentication(ctx context.Context, rawEmail string) error {
	if s.queries == nil {
		return fmt.Errorf("queries not configured")
	}
	if s.emailService == nil {
		return email.ErrSenderNotConfigured
	}

	targetEmail, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}

	rawToken, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(s.emailTokenTTL)

	if err := s.queries.InvalidateActiveSignupEmailTokensForEmail(ctx, targetEmail); err != nil {
		return fmt.Errorf("invalidate active tokens: %w", err)
	}
	if _, err := s.queries.CreateSignupEmailToken(ctx, db.CreateSignupEmailTokenParams{
		AuthSignupEmailTargetEmail: targetEmail,
		AuthSignupEmailTokenHash:   tokenHash,
		AuthSignupEmailExpiresAt:   expiresAt,
	}); err != nil {
		return fmt.Errorf("create email auth token: %w", err)
	}

	confirmURL := s.confirmURL(rawToken)
	if s.emitConsoleLink {
		fmt.Printf("[email-auth] email=%s confirm_url=%s\n", targetEmail, confirmURL)
	}
	if err := s.emailService.Send(ctx, email.Message{
		To:      []string{targetEmail},
		Subject: "Sign in to Koditon",
		Text: "Use this link to sign in to Koditon.\n\n" +
			"Continue: " + confirmURL + "\n\n" +
			"If you did not request this, you can ignore this email.",
		HTML: "<p>Use this link to sign in to Koditon.</p>" +
			"<p><a href=\"" + confirmURL + "\">Continue in Koditon</a></p>" +
			"<p>If you did not request this, you can ignore this email.</p>",
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) ConfirmAuthentication(ctx context.Context, rawToken string) (string, error) {
	if s.queries == nil {
		return "", fmt.Errorf("queries not configured")
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", ErrInvalidToken
	}
	tokenHash := hashToken(rawToken)

	emailAddr, err := s.queries.ConsumeActiveSignupEmailTokenByHash(ctx, tokenHash)
	if err != nil {
		if err != pgx.ErrNoRows {
			return "", fmt.Errorf("consume email auth token: %w", err)
		}
		status, statusErr := s.queries.GetSignupEmailTokenStatusByHash(ctx, tokenHash)
		if statusErr == pgx.ErrNoRows {
			return "", ErrInvalidToken
		}
		if statusErr != nil {
			return "", fmt.Errorf("get email auth token status: %w", statusErr)
		}
		switch strings.TrimSpace(status) {
		case "expired":
			return "", ErrTokenExpired
		case "consumed":
			return "", ErrTokenConsumed
		default:
			return "", ErrInvalidToken
		}
	}

	normalizedEmail, err := NormalizeEmail(emailAddr)
	if err != nil {
		return "", ErrInvalidEmail
	}
	rawTicket, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate auth ticket: %w", err)
	}
	ticketHash := hashToken(rawTicket)
	ticketExpiresAt := time.Now().Add(s.ticketTTL)
	if _, err := s.queries.CreateSignupTicket(ctx, db.CreateSignupTicketParams{
		AuthSignupTicketTargetEmail: normalizedEmail,
		AuthSignupTicketHash:        ticketHash,
		AuthSignupTicketExpiresAt:   ticketExpiresAt,
	}); err != nil {
		return "", fmt.Errorf("create email auth ticket: %w", err)
	}
	return rawTicket, nil
}

func (s *Service) ConsumeAuthenticationTicket(ctx context.Context, rawTicket string) (AuthenticatedEmail, error) {
	if s.queries == nil {
		return AuthenticatedEmail{}, fmt.Errorf("queries not configured")
	}
	rawTicket = strings.TrimSpace(rawTicket)
	if rawTicket == "" {
		return AuthenticatedEmail{}, ErrInvalidTicket
	}
	ticketHash := hashToken(rawTicket)
	targetEmail, err := s.queries.ConsumeActiveSignupTicketByHash(ctx, ticketHash)
	if err == nil {
		normalized, normalizeErr := emailaddr.Parse(targetEmail)
		if normalizeErr != nil {
			return AuthenticatedEmail{}, ErrInvalidEmail
		}
		return AuthenticatedEmail{address: normalized}, nil
	}
	if err != pgx.ErrNoRows {
		return AuthenticatedEmail{}, fmt.Errorf("consume email auth ticket: %w", err)
	}

	status, statusErr := s.queries.GetSignupTicketStatusByHash(ctx, ticketHash)
	if statusErr == pgx.ErrNoRows {
		return AuthenticatedEmail{}, ErrInvalidTicket
	}
	if statusErr != nil {
		return AuthenticatedEmail{}, fmt.Errorf("get email auth ticket status: %w", statusErr)
	}
	switch strings.TrimSpace(status) {
	case "expired":
		return AuthenticatedEmail{}, ErrTicketExpired
	case "consumed":
		return AuthenticatedEmail{}, ErrTicketConsumed
	default:
		return AuthenticatedEmail{}, ErrInvalidTicket
	}
}

func (s *Service) confirmURL(rawToken string) string {
	base := strings.TrimRight(s.webBaseURL, "/")
	if base != "" {
		return base + "/auth/email/confirm/" + rawToken
	}
	publicAPI := strings.TrimRight(s.publicAPIURL, "/")
	if publicAPI != "" {
		return publicAPI + "/auth/email/confirm/" + rawToken
	}
	return "/auth/email/confirm/" + rawToken
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
