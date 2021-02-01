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
serviceManager:
  credentials:
    - region: eu
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: 
saml:
  enabled: false
`, expected: "invalid config: service manager credentials: no password"},
		{in: `
serviceManager:
  credentials:
    - region: eu
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: 
      password: qwerty
saml:
  enabled: false
`, expected: "invalid config: service manager credentials: no username"},
		{in: `
serviceManager:
  credentials:
    - region: eu
      url: 
      username: sm
      password: qwerty
saml:
  enabled: false
`, expected: "invalid config: service manager credentials: no url"},
		{in: `
serviceManager:
  credentials:
    - region:
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
saml:
  enabled: false
`, expected: "invalid config: service manager credentials: no region"},
		{in: `
serviceManager:
  credentials:
    - region: aus
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
saml:
  enabled: false  
`, expected: "invalid config: service manager credentials: unsupported region: aus (eu,us supported only)"},
		{in: `
serviceManager:
  credentials:
    - region: us
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
`, expected: "invalid config: no saml"},
	}

	for _, tc := range tests {
		_, err := Load(tc.in)

		require.Error(t, err)
		require.EqualError(t, err, tc.expected)
	}
}
