package util

// BoolPtr returns pointer to given bool
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns pointer to given int
func IntPtr(val int) *int {
	return &val
}

// StringPtr returns pointer to given string
func StringPtr(str string) *string {
	return &str
}
