package passkey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	wbauthn "github.com/go-webauthn/webauthn/webauthn"
)

type Config struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

type Service struct {
	webauthn *wbauthn.WebAuthn
}

type User struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []wbauthn.Credential
}

func (u User) WebAuthnID() []byte {
	return u.ID
}

func (u User) WebAuthnName() string {
	return u.Name
}

func (u User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u User) WebAuthnCredentials() []wbauthn.Credential {
	return u.Credentials
}

func NewService(cfg Config) (*Service, error) {
	rpID := strings.TrimSpace(cfg.RPID)
	rpName := strings.TrimSpace(cfg.RPDisplayName)
	origins := make([]string, 0, len(cfg.RPOrigins))
	for _, origin := range cfg.RPOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}
	if rpID == "" {
		return nil, fmt.Errorf("passkey rpid is required")
	}
	if rpName == "" {
		return nil, fmt.Errorf("passkey rp display name is required")
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("at least one passkey origin is required")
	}

	w, err := wbauthn.New(&wbauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpName,
		RPOrigins:             origins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create webauthn config: %w", err)
	}

	return &Service{webauthn: w}, nil
}

func (s *Service) BeginDiscoverableAuthentication() (*protocol.CredentialAssertion, *wbauthn.SessionData, error) {
	assertion, session, err := s.webauthn.BeginDiscoverableLogin(
		wbauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		return nil, nil, err
	}
	return assertion, session, nil
}

func (s *Service) BeginAuthentication(user User) (*protocol.CredentialAssertion, *wbauthn.SessionData, error) {
	assertion, session, err := s.webauthn.BeginLogin(
		user,
		wbauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		return nil, nil, err
	}
	return assertion, session, nil
}

func (s *Service) BeginRegistration(user User, exclude []protocol.CredentialDescriptor) (*protocol.CredentialCreation, *wbauthn.SessionData, error) {
	options := []wbauthn.RegistrationOption{
		wbauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		}),
		wbauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		wbauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	}
	if len(exclude) > 0 {
		options = append(options, wbauthn.WithExclusions(exclude))
	}
	creation, session, err := s.webauthn.BeginRegistration(user, options...)
	if err != nil {
		return nil, nil, err
	}
	return creation, session, nil
}

func (s *Service) FinishRegistration(ctx context.Context, user User, session wbauthn.SessionData, credentialJSON []byte) (*wbauthn.Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(credentialJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return s.webauthn.FinishRegistration(user, session, req)
}

func (s *Service) FinishPasskeyLogin(ctx context.Context, session wbauthn.SessionData, credentialJSON []byte, handler wbauthn.DiscoverableUserHandler) (wbauthn.User, *wbauthn.Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(credentialJSON))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return s.webauthn.FinishPasskeyLogin(handler, session, req)
}

func MarshalSessionData(session *wbauthn.SessionData) ([]byte, error) {
	return json.Marshal(session)
}

func UnmarshalSessionData(data []byte) (wbauthn.SessionData, error) {
	var session wbauthn.SessionData
	err := json.Unmarshal(data, &session)
	return session, err
}
