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

func TestCheckGenericKymaVersion(t *testing.T) {
	var tests = []struct {
		summary    string
		constraint string
		version    string
		res        bool
	}{
		{"Check PR image", "1.21.x", "PR-1234", true},
		{"Check Master image", "1.21.x", "master-abcd2", true},
		{"Check Kyma image 1.21.1", "<1.21.x", "1.21.1", true},
		{"Check Kyma image 1.21.0", "<1.21.x", "1.21.0", true},
		{"Check Kyma image 1.20.0", "<1.21.x", "1.20.0", false},
		{"Check Kyma image 1.20.0-rc4", "<1.21.x", "1.20.0-rc4", true}, // mapped to true, like "PR*" or "master*"
		{"Check Kyma image 1.21.1", "<1.21.2", "1.21.1", false},
		{"Check Kyma image 1.19.1", "<1.20.x", "1.19.1", false},
	}
	for _, tt := range tests {
		t.Log(tt.summary)
		res, err := isKymaVersionAtLeast(tt.constraint, tt.version)
		require.NoError(t, err)
		require.Equal(t, tt.res, res)
	}
}

func TestCheckKymaVersionAtLeast_1_21(t *testing.T) {
	var tests = []struct {
		summary string
		version string
		res     bool
	}{
		{"Check PR image", "PR-1234", true},
		{"Check Master image", "master-abcd2", true},
		{"Check Kyma image 1.21.1", "1.21.1", true},
		{"Check Kyma image 1.21.0", "1.21.0", true},
		{"Check Kyma image 1.20.0", "1.20.0", false},
		{"Check Kyma image 1.21.1", "1.21.1", true},
		{"Check Kyma image 1.19.1", "1.19.1", false},
	}
	for _, tt := range tests {
		t.Log(tt.summary)
		res, err := IsKymaVersionAtLeast_1_21(tt.version)
		require.NoError(t, err)
		require.Equal(t, tt.res, res)
	}
}
