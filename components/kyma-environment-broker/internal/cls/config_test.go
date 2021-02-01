package cls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadEmptyString(t *testing.T) {
	var in string
	_, err := Load(in)

	expected := "invalid config: no service manager credentials"
	require.Error(t, err)
	require.EqualError(t, err, expected)
}

func TestLoadInvalidCredentials(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{in: `
serviceManagerCredentials:
  regions:
    eu:
      url: http://service-manager.com
      username: sm
      password: 
`, expected: "invalid config: no password"},
		{in: `
serviceManagerCredentials:
  regions:
    eu:
      url: http://service-manager.com
      username: 
      password: qwerty
`, expected: "invalid config: no username"},
		{in: `
serviceManagerCredentials:
  regions:
    eu:
      url: 
      username: sm
      password: qwerty
`, expected: "invalid config: no url"},
	}

	for _, tc := range tests {
		_, err := Load(tc.in)

		require.Error(t, err)
		require.EqualError(t, err, tc.expected)
	}
}
