package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/stretchr/testify/require"
)

func TestServiceManagerClient(t *testing.T) {
	tests := []struct {
		summary                    string
		givenServiceManagerRegions []string
		givenSKRRegion             *string
		expectedError              string
	}{
		{
			summary:                    "unsupported skr region",
			givenServiceManagerRegions: []string{"eu", "us"},
			givenSKRRegion:             stringPtr("westeurope42"),
			expectedError:              "unsupported region: westeurope42",
		},
		{
			summary:                    "no matching service manager credentials",
			givenServiceManagerRegions: []string{"us"},
			givenSKRRegion:             stringPtr("westeurope"),
			expectedError:              "unable find credentials for the region: eu",
		},
		{
			summary:                    "happy path",
			givenServiceManagerRegions: []string{"eu", "us"},
			givenSKRRegion:             stringPtr("westeurope"),
		},
		{
			summary:                    "happy path (default service manager region)",
			givenServiceManagerRegions: []string{"eu", "us"},
			givenSKRRegion:             nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			config := &ServiceManagerConfig{}
			for _, r := range tc.givenServiceManagerRegions {
				config.Credentials = append(config.Credentials, &ServiceManagerCredentials{Region: Region(r)})
			}

			operation := &internal.ProvisioningOperation{
				Operation: internal.Operation{
					ProvisioningParameters: internal.ProvisioningParameters{
						Parameters: internal.ProvisioningParametersDTO{
							Region: tc.givenSKRRegion,
						},
					},
				},
				SMClientFactory: servicemanager.NewFakeServiceManagerClientFactory(nil, nil),
			}

			client, err := ServiceManagerClient(config, operation)

			if len(tc.expectedError) > 0 {
				require.EqualError(t, err, tc.expectedError)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
