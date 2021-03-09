package cls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckKymaVersionPR(t *testing.T) {
	var tests = []struct {
		summary string
		version string
		res     bool
	}{
		{"Check PR image", "PR-1234", true},
		{"Check Master image", "master-abcd2", true},
		{"Check Kyma image 1.21.1", "1.21.1", true},
		{"Check Kyma image 1.19.1", "1.19.1", false},
	}
	for _, tt := range tests {
		t.Log(tt.summary)
		res, err := IsKymaVersionAtLeast_1_20(tt.version)
		require.NoError(t, err)
		require.Equal(t, tt.res, res)
	}
}
