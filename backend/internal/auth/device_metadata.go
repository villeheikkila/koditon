package auth

import (
	"context"
	"fmt"
	"strings"

	db "koditon-go/internal/db"
	"koditon-go/internal/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) updateDeviceMetadata(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
	params createSessionParams,
) error {
	if deviceID == uuid.Nil {
		return nil
	}

	qtx := s.queries.WithTx(tx)
	if err := qtx.UpdateDeviceMetadata(ctx, db.UpdateDeviceMetadataParams{
		UserDeviceUuid:       util.UUIDToPg(deviceID),
		UserDeviceName:       nullableString(params.DeviceName),
		UserDeviceOs:         nullableString(params.DeviceOS),
		UserDeviceModel:      nullableString(params.DeviceModel),
		UserDeviceLocale:     nullableString(params.DeviceLocale),
		UserDeviceTimeZone:   nullableString(params.DeviceTimeZone),
		UserDeviceAppVersion: nullableString(params.DeviceAppVersion),
	}); err != nil {
		return fmt.Errorf("update device metadata: %w", err)
	}
	return nil
}

func (s *Service) updateSessionMetadata(
	ctx context.Context,
	tx pgx.Tx,
	sessionID uuid.UUID,
	params createSessionParams,
) error {
	qtx := s.queries.WithTx(tx)
	location := s.resolveSessionLocation(ctx, params)
	if err := qtx.UpdateSessionMetadata(ctx, db.UpdateSessionMetadataParams{
		DeviceSessionUuid:                util.UUIDToPg(sessionID),
		DeviceSessionDeviceName:          nullableString(params.DeviceName),
		DeviceSessionDeviceOs:            nullableString(params.DeviceOS),
		DeviceSessionDeviceModel:         nullableString(params.DeviceModel),
		DeviceSessionAppVersion:          nullableString(params.DeviceAppVersion),
		DeviceSessionLocale:              nullableString(params.DeviceLocale),
		DeviceSessionTimeZone:            nullableString(params.DeviceTimeZone),
		DeviceSessionLocationCity:        nullableString(location.City),
		DeviceSessionLocationRegion:      nullableString(location.Region),
		DeviceSessionLocationCountryCode: nullableString(location.CountryCode),
		DeviceSessionLocationSource:      nullableString(location.Source),
	}); err != nil {
		return fmt.Errorf("update session metadata: %w", err)
	}
	return nil
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
