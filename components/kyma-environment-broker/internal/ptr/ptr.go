package ptr

import (
	"fmt"
	"time"
)

func Bool(in bool) *bool {
	return &in
}

func BoolAsString(in *bool) string {
	if in == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", *in)
}

func String(str string) *string {
	return &str
}

func ToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func Integer(in int) *int {
	return &in
}

func Time(in time.Time) *time.Time {
	return &in
}
