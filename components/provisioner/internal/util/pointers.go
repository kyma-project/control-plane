package util

import (
	"k8s.io/apimachinery/pkg/util/intstr"
)

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

// IntOrStringPtr returns pointer to given int or string
func IntOrStringPtr(intOrStr intstr.IntOrString) *intstr.IntOrString {
	return &intOrStr
}

// UnwrapInt returns int value from pointer
func UnwrapInt(intPtr *int) int {
	if intPtr == nil {
		return 0
	}
	return *intPtr
}

// UnwrapStr returns string value from pointer
func UnwrapStr(strPtr *string) string {
	if strPtr == nil {
		return ""
	}
	return *strPtr
}

// UnwrapBoolOrDefault returns bool value from pointer or if pointer is nil returns default value
func UnwrapBoolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

// UnwrapIntOrDefault returns int value from pointer or if pointer is nil returns default value
func UnwrapIntOrDefault(ptr *int, def int) int {
	if ptr == nil {
		return def
	}
	return *ptr
}

// UnwrapStrOrDefault returns string value from pointer or if pointer is nil returns default value
func UnwrapStrOrDefault(ptr *string, def string) string {
	if ptr == nil {
		return def
	}
	return *ptr
}

// DefaultStrIfNil returns default string pointer if passed pointer is nil
func DefaultStrIfNil(ptr *string, def *string) *string {
	if ptr == nil {
		return def
	}
	return ptr
}

// DefaultIntIfNil returns default int pointer if passed pointer is nil
func DefaultIntIfNil(ptr *int, def *int) *int {
	if ptr == nil {
		return def
	}
	return ptr
}

// DefaultBoolIfNil returns default bool pointer if passed pointer is nil
func DefaultBoolIfNil(ptr *bool, def *bool) *bool {
	if ptr == nil {
		return def
	}
	return ptr
}
