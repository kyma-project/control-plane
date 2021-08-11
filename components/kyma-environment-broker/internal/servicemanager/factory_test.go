package servicemanager

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFactory_ForCustomerCredentials_ModeAlways(t *testing.T) {
	// given
	factory := NewClientFactory(Config{
		OverrideMode:                     "Always",
		URL:                              "http://default.url",
		Password:                         "default_password",
		Username:                         "default_username",
		SubaccountWithRequestCredentials: "special_subaccountID",
	})

	for tn, tc := range map[string]struct {
		givenRequest        RequestContext
		expectedCredentials Credentials
	}{
		"regular SubAccount": {
			givenRequest: RequestContext{
				SubaccountID: "regular_saID",
				Credentials:  nil,
			},
			expectedCredentials: Credentials{
				Password: "default_password",
				Username: "default_username",
				URL:      "http://default.url",
			},
		},
		"special SubAccount": {
			givenRequest: RequestContext{
				SubaccountID: "special_subaccountID",
				Credentials: &Credentials{
					Password: "p",
					Username: "u",
					URL:      "http://url",
				},
			},
			expectedCredentials: Credentials{
				Password: "p",
				Username: "u",
				URL:      "http://url",
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// when
			cli, err := factory.ForCustomerCredentials(tc.givenRequest, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCredentials, cli.(*client).creds)
		})
	}
}
