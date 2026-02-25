package util

func ToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func Int32Ptr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

//go:fix inline
func Int64Ptr(i int64) *int64 {
	return new(i)
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
