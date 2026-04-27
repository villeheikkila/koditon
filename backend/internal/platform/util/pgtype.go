package util

func Int32Ptr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func FloatToInt32Ptr(f *float64) *int32 {
	if f == nil {
		return nil
	}
	v := int32(*f)
	return &v
}
