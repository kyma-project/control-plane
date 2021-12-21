package timestamp

import (
	"fmt"
	"strings"
	"time"
)

// Parse recognizes a given timestamp and returns the right Time object or error. Proper formats:
// yyyy/mm/dd hh:mm:ss
// hh:mm:ss yyyy/mm/dd
// yyyy/mm/dd
// hh:mm:ss
// The late argument set to true cause adding "23:59:59" time when given timestamp is in format "yyyy/mm/dd" instead of "00:00:00"
func Parse(timestamp string, late bool) (time.Time, error) {
	t, err := formatTimestamp(timestamp, late)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse("2006/01/02 15:04:05", t)
}

func formatTimestamp(timestamp string, late bool) (string, error) {
	t := []byte(timestamp)
	for i := range t {
		if t[i] != ':' && t[i] != '/' && t[i] != ' ' {
			t[i] = 'd'
		}
	}

	switch string(t) {
	case "dddd/dd/dd":
		t := "00:00:00"
		if late {
			t = "23:59:59"
		}
		return fmt.Sprintf("%s %s", timestamp, t), nil
	case "dd:dd:dd":
		y, m, d := time.Now().Date()
		return fmt.Sprintf("%04d/%02d/%02d %s", y, m, d, timestamp), nil
	case "dddd/dd/dd dd:dd:dd":
		return timestamp, nil
	case "dd:dd:dd dddd/dd/dd":
		s := strings.Split(timestamp, " ")
		return fmt.Sprintf("%s %s", s[1], s[0]), nil
	}

	return "", fmt.Errorf("cannot match right pattern for the given timestamp: %s", timestamp)
}
