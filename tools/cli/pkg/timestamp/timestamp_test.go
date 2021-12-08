package timestamp

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	y, m, d := time.Now().Date()
	t1, err := time.Parse("2006/01/02 15:04:05", "2006/01/02 00:00:00")
	require.NoError(t, err)
	t1late, err := time.Parse("2006/01/02 15:04:05", "2006/01/02 23:59:59")
	require.NoError(t, err)
	t2Value := fmt.Sprintf("%04d/%02d/%02d %s", y, m, d, "12:02:46")
	t2ValueReverted := fmt.Sprintf("%s %04d/%02d/%02d", "12:02:46", y, m, d)
	t2, err := time.Parse("2006/01/02 15:04:05", t2Value)
	require.NoError(t, err)

	tests := []struct {
		name      string
		timestamp string
		want      time.Time
		wantLate  bool
		wantErr   bool
	}{
		{
			name:    "empty input",
			wantErr: true,
		},
		{
			name:      "date",
			timestamp: "2006/01/02",
			want:      t1,
			wantErr:   false,
		},
		{
			name:      "late date",
			timestamp: "2006/01/02",
			want:      t1late,
			wantLate:  true,
			wantErr:   false,
		},
		{
			name:      "time",
			timestamp: "12:02:46",
			want:      t2,
			wantErr:   false,
		},
		{
			name:      "late time",
			timestamp: "12:02:46",
			want:      t2,
			wantLate:  true,
			wantErr:   false,
		},
		{
			name:      "date and time",
			timestamp: t2Value,
			want:      t2,
			wantErr:   false,
		},
		{
			name:      "late date and time",
			timestamp: t2Value,
			want:      t2,
			wantLate:  true,
			wantErr:   false,
		},
		{
			name:      "time and date",
			timestamp: t2ValueReverted,
			want:      t2,
			wantErr:   false,
		},
		{
			name:      "late time and date",
			timestamp: t2ValueReverted,
			want:      t2,
			wantLate:  true,
			wantErr:   false,
		},
		{
			name:      "wront format",
			timestamp: "2015-02-21 12:00:12.000",
			wantErr:   true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.timestamp, tt.wantLate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
