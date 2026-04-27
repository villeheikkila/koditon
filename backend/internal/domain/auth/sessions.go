package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ListSessionItem struct {
	SessionID            uuid.UUID
	Provider             string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	LastRefreshedAt      *time.Time
	ExpiresAt            *time.Time
	DeviceName           *string
	DeviceOS             *string
	DeviceModel          *string
	DeviceLocale         *string
	DeviceTimeZone       *string
	DeviceLocationCity   *string
	DeviceLocationRegion *string
	DeviceCountryCode    *string
	DeviceLocationSource *string
	AppVersion           *string
	LastSeenAt           *time.Time
	UserAgent            *string
	IsCurrent            bool
}

func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID uuid.UUID) ([]ListSessionItem, error) {
	rows, err := s.queries.GetSessionsByUserID(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	items := make([]ListSessionItem, 0, len(rows))
	for _, row := range rows {
		var lastRefreshedAt *time.Time
		if row.DeviceSessionRefreshedAt != nil {
			lastRefreshedAt = row.DeviceSessionRefreshedAt
		}

		var expiresAt *time.Time
		if row.DeviceSessionNotAfter != nil {
			expiresAt = row.DeviceSessionNotAfter
		}

		lastSeenAt := row.UserDeviceLastSeenAt

		items = append(items, ListSessionItem{
			SessionID:            row.DeviceSessionUuid,
			Provider:             row.DeviceSessionProvider,
			CreatedAt:            row.DeviceSessionCreatedAt,
			UpdatedAt:            row.DeviceSessionUpdatedAt,
			LastRefreshedAt:      lastRefreshedAt,
			ExpiresAt:            expiresAt,
			DeviceName:           row.DeviceName,
			DeviceOS:             row.DeviceOs,
			DeviceModel:          row.DeviceModel,
			DeviceLocale:         row.DeviceLocale,
			DeviceTimeZone:       row.DeviceTimeZone,
			DeviceLocationCity:   row.DeviceSessionLocationCity,
			DeviceLocationRegion: row.DeviceSessionLocationRegion,
			DeviceCountryCode:    row.DeviceSessionLocationCountryCode,
			DeviceLocationSource: row.DeviceSessionLocationSource,
			AppVersion:           row.DeviceAppVersion,
			LastSeenAt:           &lastSeenAt,
			UserAgent:            row.DeviceSessionUserAgent,
			IsCurrent:            row.DeviceSessionUuid == currentSessionID,
		})
	}

	return items, nil
}
