package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	db "koditon-go/internal/db"
	"koditon-go/internal/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ConnectedAppItem struct {
	ClientID     string
	DisplayName  string
	LogoURL      string
	IsFirstParty bool
	Scopes       []string
	ConnectedAt  time.Time
	LastUsedAt   time.Time
}

func (s *Service) ListConnectedApps(ctx context.Context, userID uuid.UUID) ([]ConnectedAppItem, error) {
	rows, err := s.queries.ListOAuthAppConnectionsByUserID(ctx, util.UUIDToPg(userID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("list connected apps: %w", err)
	}

	items := make([]ConnectedAppItem, 0, len(rows))
	for _, row := range rows {
		clientID := strings.TrimSpace(row.OauthClientID)
		if clientID == "" {
			continue
		}

		metadata, hasMetadata := OAuthClientMetadataForID(clientID)
		if hasMetadata && !metadata.ShowInConnectedApps {
			continue
		}

		displayName := ""
		if row.OauthDynamicClientName != nil {
			displayName = strings.TrimSpace(*row.OauthDynamicClientName)
		}
		if displayName == "" && hasMetadata {
			displayName = strings.TrimSpace(metadata.DisplayName)
		}
		if displayName == "" {
			displayName = clientID
		}
		logoURL := ""
		if hasMetadata {
			logoURL = strings.TrimSpace(metadata.LogoURL)
		}
		if logoURL == "" {
			logoURL = s.dynamicClientLogoURL(ctx, clientID)
		}

		items = append(items, ConnectedAppItem{
			ClientID:     clientID,
			DisplayName:  displayName,
			LogoURL:      logoURL,
			IsFirstParty: hasMetadata && metadata.IsFirstParty,
			Scopes:       append([]string(nil), row.Scopes...),
			ConnectedAt:  row.ConnectedAt.Time,
			LastUsedAt:   row.LastUsedAt.Time,
		})
	}

	return items, nil
}

func (s *Service) RevokeConnectedApp(ctx context.Context, userID uuid.UUID, clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}

	_, err := s.queries.RevokeAllOAuthRefreshTokensByUserIDAndClientID(ctx, db.RevokeAllOAuthRefreshTokensByUserIDAndClientIDParams{
		UserUuid:      util.UUIDToPg(userID),
		OauthClientID: &clientID,
	})
	if err != nil {
		return fmt.Errorf("revoke connected app: %w", err)
	}

	s.emitTokenEvent(ctx, tokenEvent{
		Name:      tokenEventRevoked,
		AuthType:  string(AccessTokenKindOAuth),
		ClientID:  clientID,
		UserID:    userID,
		TokenType: "refresh",
	})
	return nil
}

func (s *Service) dynamicClientLogoURL(ctx context.Context, clientID string) string {
	row, err := s.queries.GetOAuthDynamicClientByID(ctx, &clientID)
	if err != nil {
		return ""
	}
	var metadata struct {
		LogoURI string `json:"logo_uri"`
	}
	if err := json.Unmarshal(row.OauthDynamicClientMetadata, &metadata); err != nil {
		return ""
	}
	return strings.TrimSpace(metadata.LogoURI)
}
