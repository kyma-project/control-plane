package ptr

import "time"

func Bool(in bool) *bool {
	return &in
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
