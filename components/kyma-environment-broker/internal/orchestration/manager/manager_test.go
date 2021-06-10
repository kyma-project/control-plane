package manager

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
)

func Test_convertSliceOfDaysToMap(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		// given
		days := []time.Weekday{time.Tuesday, time.Thursday}

		// when
		m := orchestration.ConvertSliceOfDaysToMap(days)

		// then
		for _, day := range days {
			_, doesExist := m[day]
			if !doesExist {
				t.Errorf("convertSliceOfDaysToMap(\"time.Tuesday, time.Thursday\") failed")
			}
		}
	})
}

func Test_firstAvailableDay(t *testing.T) {
	t.Run("current day is Wednesday and available days are {Tuesday, Thursday}", func(t *testing.T) {
		// given
		m := make(map[time.Weekday]bool)
		m[time.Tuesday] = true
		m[time.Thursday] = true

		// when
		result := orchestration.FirstAvailableDay(3, m)

		// then
		if result != 4 {
			t.Errorf("firstAvailableDay(\"3, m\") failed, expected %v, got %v", "4", result)
		}
	})
}
