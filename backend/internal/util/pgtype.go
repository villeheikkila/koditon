package util

import "github.com/jackc/pgx/v5/pgtype"

func ToInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

func ToInt8(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}

func ToFloat8(f *float64) pgtype.Float8 {
	if f == nil {
		return pgtype.Float8{Valid: false}
	}
	return pgtype.Float8{Float64: *f, Valid: true}
}

func ToBoolean(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func FloatToInt4(f *float64) pgtype.Int4 {
	if f == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*f), Valid: true}
}

func ToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Pointer-based helpers for consolidated sqlc types

func Int32Ptr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func Float64Ptr(f *float64) *float64 {
	return f
}

func BoolPtr(b *bool) *bool {
	return b
}

func FloatToInt32Ptr(f *float64) *int32 {
	if f == nil {
		return nil
	}
	v := int32(*f)
	return &v
}
