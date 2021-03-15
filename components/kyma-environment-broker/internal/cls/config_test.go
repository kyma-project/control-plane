package cls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadEmptyString(t *testing.T) {
	var in string
	_, err := Load(in)

	expected := "invalid config: no Service Manager credentials"
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
  initiated: true
`, expected: "invalid config: while validating Service Manager credentials: no password"},
		{in: `
serviceManager:
  credentials:
    - region: eu
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: 
      password: qwerty
saml:
  initiated: true
`, expected: "invalid config: while validating Service Manager credentials: no username"},
		{in: `
serviceManager:
  credentials:
    - region: eu
      url: 
      username: sm
      password: qwerty
saml:
  initiated: true
`, expected: "invalid config: while validating Service Manager credentials: no URL"},
		{in: `
serviceManager:
  credentials:
    - region:
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
saml:
  initiated: true
`, expected: "invalid config: while validating Service Manager credentials: no region"},
		{in: `
serviceManager:
  credentials:
    - region: aus
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
saml:
  initiated: true  
`, expected: "invalid config: while validating Service Manager credentials: unsupported region: aus (eu,us supported only)"},
		{in: `
serviceManager:
  credentials:
    - region: us
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
`, expected: "invalid config: no SAML"},
	}

	for _, tc := range tests {
		// given
		// when
		_, err := Load(tc.in)

		// then
		require.Error(t, err)
		require.EqualError(t, err, tc.expected)
	}
}

func TestLoadDefaultParams(t *testing.T) {
	in := `
serviceManager:
  credentials:
    - region: us
      url: https://service-manager.cfapps.sap.hana.ondemand.com
      username: sm
      password: qwerty
saml:
  initiated: true  
`
	// given
	// when
	config, err := Load(in)

	// then
	require.NoError(t, err)
	require.Equal(t, 7, config.RetentionPeriod)
	require.Equal(t, 2, config.MaxDataInstances)
	require.Equal(t, 2, config.MaxIngestInstances)
}
