package schema

import (
	"context"
	"fmt"
)

const RequiredVersion = 11

type DB interface {
	GetSchemaVersion(ctx context.Context) (int32, error)
}

func Check(ctx context.Context, db DB) error {
	version, err := CurrentVersion(ctx, db)
	if err != nil {
		return err
	}
	if version < RequiredVersion {
		return fmt.Errorf("schema version %d is below required version %d", version, RequiredVersion)
	}
	return nil
}

func CurrentVersion(ctx context.Context, db DB) (int, error) {
	version, err := db.GetSchemaVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return int(version), nil
}
