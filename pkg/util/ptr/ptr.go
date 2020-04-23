package ptr

func Bool(value bool) *bool {
	return &value
}

func Int32(value int32) *int32 {
	return &value
}

func Uint64(value uint64) *uint64 {
	return &value
}

func Int64(value int64) *int64 {
	return &value
}

func String(value string) *string {
	return &value
}
