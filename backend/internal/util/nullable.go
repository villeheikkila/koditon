package util

import (
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

type Nullable[T any] struct {
	Value T
	Valid bool
}

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Value)
}

func NewNullable[T any](value T) Nullable[T] {
	return Nullable[T]{
		Value: value,
		Valid: true,
	}
}

func FromSQLNullString(ns sql.NullString) Nullable[string] {
	return Nullable[string]{
		Value: ns.String,
		Valid: ns.Valid,
	}
}

func FromSQLNullInt32(ni sql.NullInt32) Nullable[int32] {
	return Nullable[int32]{
		Value: ni.Int32,
		Valid: ni.Valid,
	}
}

func FromSQLNullInt64(ni sql.NullInt64) Nullable[int64] {
	return Nullable[int64]{
		Value: ni.Int64,
		Valid: ni.Valid,
	}
}

func FromSQLNullFloat64(nf sql.NullFloat64) Nullable[float64] {
	return Nullable[float64]{
		Value: nf.Float64,
		Valid: nf.Valid,
	}
}

func FromSQLNullBool(nb sql.NullBool) Nullable[bool] {
	return Nullable[bool]{
		Value: nb.Bool,
		Valid: nb.Valid,
	}
}

func FromPgText(pt pgtype.Text) Nullable[string] {
	return Nullable[string]{
		Value: pt.String,
		Valid: pt.Valid,
	}
}

func FromPgInt4(pi pgtype.Int4) Nullable[int32] {
	return Nullable[int32]{
		Value: pi.Int32,
		Valid: pi.Valid,
	}
}

func FromPgInt8(pi pgtype.Int8) Nullable[int64] {
	return Nullable[int64]{
		Value: pi.Int64,
		Valid: pi.Valid,
	}
}

func FromPgFloat8(pf pgtype.Float8) Nullable[float64] {
	return Nullable[float64]{
		Value: pf.Float64,
		Valid: pf.Valid,
	}
}

func FromPgBool(pb pgtype.Bool) Nullable[bool] {
	return Nullable[bool]{
		Value: pb.Bool,
		Valid: pb.Valid,
	}
}
