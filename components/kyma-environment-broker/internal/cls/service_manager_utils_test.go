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
		expectedError    string
	}{
		{
			summary:        "unsupported skr region",
			givenSKRRegion: stringPtr("westeurope42"),
			expectedError:  "unsupported region: westeurope42",
		},
		{
			summary:          "happy path",
			givenSKRRegion:   stringPtr("westeurope"),
			expectedSMRegion: "eu",
		},
		{
			summary:          "happy path (default service manager region)",
			givenSKRRegion:   nil,
			expectedSMRegion: "eu",
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			smRegion, err := DetermineServiceManagerRegion(tc.givenSKRRegion)

			if len(tc.expectedError) > 0 {
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedSMRegion, smRegion)
			}
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
				&ServiceManagerCredentials{
					Region:   "us",
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
				&ServiceManagerCredentials{
					Region:   "eu",
					URL:      "eu.service-manager.com",
					Username: "john.doe",
					Password: "qwerty",
				},
				&ServiceManagerCredentials{
					Region:   "us",
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
			config := &ServiceManagerConfig{
				Credentials: tc.givenCredentials,
			}

			credentials, err := FindCredentials(config, tc.givenSMRegion)

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
