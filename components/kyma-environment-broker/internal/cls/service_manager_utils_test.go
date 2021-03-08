package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/stretchr/testify/require"
)

func TestDetermineServiceManagerRegion(t *testing.T) {
	tests := []struct {
		summary          string
		givenSKRRegion   *string
		expectedSMRegion string
	}{
		{
			summary:          "unsupported skr region",
			givenSKRRegion:   stringPtr("westeurope42"),
			expectedSMRegion: RegionEurope,
		},
		{
			summary:          "unknown skr region",
			givenSKRRegion:   nil,
			expectedSMRegion: RegionEurope,
		},
		{
			summary:          "supported europian region",
			givenSKRRegion:   stringPtr("westeurope"),
			expectedSMRegion: RegionEurope,
		},
		{
			summary:          "supported us region",
			givenSKRRegion:   stringPtr("eastus"),
			expectedSMRegion: RegionUS,
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			// given
			// when
			smRegion := DetermineServiceManagerRegion(tc.givenSKRRegion)

			// then
			require.Equal(t, tc.expectedSMRegion, smRegion)
		})
	}
}

func TestFindCredentials(t *testing.T) {
	tests := []struct {
		summary             string
		givenCredentials    []*ServiceManagerCredentials
		givenSMRegion       string
		expectedCredentials *servicemanager.Credentials
		expectedError       string
	}{
		{
			summary: "no matching service manager credentials",
			givenCredentials: []*ServiceManagerCredentials{
				{
					Region:   RegionUS,
					URL:      "us.service-manager.com",
					Username: "john.doe",
					Password: "qwerty",
				},
			},
			givenSMRegion: "eu",
			expectedError: "unable to find credentials for region: eu",
		},
		{
			summary: "happy path",
			givenCredentials: []*ServiceManagerCredentials{
				{
					Region:   RegionEurope,
					URL:      "eu.service-manager.com",
					Username: "john.doe",
					Password: "qwerty",
				},
				{
					Region:   RegionUS,
					URL:      "us.service-manager.com",
					Username: "john.doe",
					Password: "qwerty",
				},
			},
			givenSMRegion: "eu",
			expectedCredentials: &servicemanager.Credentials{
				URL:      "eu.service-manager.com",
				Username: "john.doe",
				Password: "qwerty",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			// given
			config := &ServiceManagerConfig{
				Credentials: tc.givenCredentials,
			}

			// when
			credentials, err := FindCredentials(config, tc.givenSMRegion)

			// then
			if len(tc.expectedError) > 0 {
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCredentials, credentials)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
