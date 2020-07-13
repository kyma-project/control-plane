package util

import (
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func StringPtr(str string) *string {
	return &str
}

func IntPtr(val int) *int {
	return &val
}

func BoolPtr(b bool) *bool {
	return &b
}

func BoolFromPtr(val *bool) bool {
	if val == nil {
		return false
	}

	return *val
}

func BoolFromPtrOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func IntOrStrPtr(intOrStr intstr.IntOrString) *intstr.IntOrString {
	return &intOrStr
}

func TimePtr(time time.Time) *time.Time {
	return &time
}

func UnwrapStr(strPtr *string) string {
	if strPtr == nil {
		return ""
	}
	return *strPtr
}

func UnwrapInt(intPtr *int) int {
	if intPtr == nil {
		return 0
	}
	return *intPtr
}

func UnwrapString(strPtr *string, defaultValue string) string {
	if strPtr == nil {
		return defaultValue
	}
	return *strPtr
}

func UnwrapBool(boolPtr *bool, defaultValue bool) bool {
	if boolPtr == nil {
		return defaultValue
	}
	return *boolPtr
}

func UnwrapIntOrGiveValue(intPtr *int, defaultValue int) int {
	if intPtr == nil {
		return defaultValue
	}
	return *intPtr
}
