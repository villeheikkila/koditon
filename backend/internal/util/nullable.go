package util

import (
	"database/sql"
	"encoding/json"
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

func FromPtr[T any](p *T) Nullable[T] {
	if p == nil {
		return Nullable[T]{}
	}
	return Nullable[T]{Value: *p, Valid: true}
}
