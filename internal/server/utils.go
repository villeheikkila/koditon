package server

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFromPg(value pgtype.UUID) (uuid.UUID, bool) {
	if !value.Valid {
		return uuid.UUID{}, false
	}
	return uuid.UUID(value.Bytes), true
}

func stringPtrFromPg(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	str := value.String
	return &str
}

func timeFromPg(value pgtype.Timestamptz) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}
