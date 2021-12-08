package timestamp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	dateLayout = `\d\d\d\d/\d\d/\d\d`
	timeLayout = `\d\d:\d\d:\d\d`
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
	ok, err := regexp.MatchString(fmt.Sprintf("%s$", dateLayout), timestamp)
	if ok && len(timestamp) == 10 {
		t := "00:00:00"
		if late {
			t = "23:59:59"
		}
		return fmt.Sprintf("%s %s", timestamp, t), err
	}

	ok, err = regexp.MatchString(fmt.Sprintf("%s$", timeLayout), timestamp)
	if ok && len(timestamp) == 8 {
		y, m, d := time.Now().Date()
		return fmt.Sprintf("%04d/%02d/%02d %s", y, m, d, timestamp), err
	}

	ok, err = regexp.MatchString(fmt.Sprintf("%s %s$", dateLayout, timeLayout), timestamp)
	if ok && len(timestamp) == 19 {
		return timestamp, err
	}

	ok, err = regexp.MatchString(fmt.Sprintf("%s %s$", timeLayout, dateLayout), timestamp)
	if ok && len(timestamp) == 19 {
		s := strings.Split(timestamp, " ")
		return fmt.Sprintf("%s %s", s[1], s[0]), err
	}

	return "", fmt.Errorf("cannot match right pattern for the given timestamp: %s", timestamp)
}
