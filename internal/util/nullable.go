package util

import (
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// Nullable is a generic type for handling optional values in JSON responses.
// It omits the field entirely when the value is null/invalid.
type Nullable[T any] struct {
	Value T
	Valid bool
}

// MarshalJSON implements json.Marshaler for Nullable.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

// UnmarshalJSON implements json.Unmarshaler for Nullable.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Value)
}

// NewNullable creates a Nullable with a valid value.
func NewNullable[T any](value T) Nullable[T] {
	return Nullable[T]{
		Value: value,
		Valid: true,
	}
}

// FromSQLNullString converts sql.NullString to Nullable[string].
func FromSQLNullString(ns sql.NullString) Nullable[string] {
	return Nullable[string]{
		Value: ns.String,
		Valid: ns.Valid,
	}
}

// FromSQLNullInt32 converts sql.NullInt32 to Nullable[int32].
func FromSQLNullInt32(ni sql.NullInt32) Nullable[int32] {
	return Nullable[int32]{
		Value: ni.Int32,
		Valid: ni.Valid,
	}
}

// FromSQLNullInt64 converts sql.NullInt64 to Nullable[int64].
func FromSQLNullInt64(ni sql.NullInt64) Nullable[int64] {
	return Nullable[int64]{
		Value: ni.Int64,
		Valid: ni.Valid,
	}
}

// FromSQLNullFloat64 converts sql.NullFloat64 to Nullable[float64].
func FromSQLNullFloat64(nf sql.NullFloat64) Nullable[float64] {
	return Nullable[float64]{
		Value: nf.Float64,
		Valid: nf.Valid,
	}
}

// FromSQLNullBool converts sql.NullBool to Nullable[bool].
func FromSQLNullBool(nb sql.NullBool) Nullable[bool] {
	return Nullable[bool]{
		Value: nb.Bool,
		Valid: nb.Valid,
	}
}

// FromPgText converts pgtype.Text to Nullable[string].
func FromPgText(pt pgtype.Text) Nullable[string] {
	return Nullable[string]{
		Value: pt.String,
		Valid: pt.Valid,
	}
}

// FromPgInt4 converts pgtype.Int4 to Nullable[int32].
func FromPgInt4(pi pgtype.Int4) Nullable[int32] {
	return Nullable[int32]{
		Value: pi.Int32,
		Valid: pi.Valid,
	}
}

// FromPgInt8 converts pgtype.Int8 to Nullable[int64].
func FromPgInt8(pi pgtype.Int8) Nullable[int64] {
	return Nullable[int64]{
		Value: pi.Int64,
		Valid: pi.Valid,
	}
}

// FromPgFloat8 converts pgtype.Float8 to Nullable[float64].
func FromPgFloat8(pf pgtype.Float8) Nullable[float64] {
	return Nullable[float64]{
		Value: pf.Float64,
		Valid: pf.Valid,
	}
}

// FromPgBool converts pgtype.Bool to Nullable[bool].
func FromPgBool(pb pgtype.Bool) Nullable[bool] {
	return Nullable[bool]{
		Value: pb.Bool,
		Valid: pb.Valid,
	}
}
