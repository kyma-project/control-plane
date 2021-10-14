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
		OverrideMode:                     SMOverrideModeAlways,
		URL:                              "http://default.url",
		Password:                         "default_password",
		Username:                         "default_username",
		SubaccountWithRequestCredentials: "special_subaccountID",
	})

	for tn, tc := range map[string]struct {
		givenRequest        RequestContext
		expectedCredentials *Credentials
	}{
		"regular SubAccount": {
			givenRequest: RequestContext{
				SubaccountID:           "regular_saID",
				Credentials:            nil,
				BTPOperatorCredentials: nil,
			},
			expectedCredentials: &Credentials{
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
			expectedCredentials: &Credentials{
				Password: "p",
				Username: "u",
				URL:      "http://url",
			},
		},
		"with BTP Operator credentials": {
			givenRequest: RequestContext{
				SubaccountID: "regular_saID",
				Credentials: &Credentials{
					Password: "p",
					Username: "u",
					URL:      "http://url",
				},
				BTPOperatorCredentials: &BTPOperatorCredentials{
					ClientID:     "c-id",
					ClientSecret: "c-s",
					TokenURL:     "http://btp-url",
					ClusterID:    "cl-id",
				},
			},
			expectedCredentials: &Credentials{
				Password: "default_password",
				Username: "default_username",
				URL:      "http://default.url",
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

func TestClientFactory_ForCustomerCredentials_ModeWhenNotSentInRequest(t *testing.T) {
	// given
	factory := NewClientFactory(Config{
		OverrideMode:                     SMOverrideModeWhenNotSentInRequest,
		URL:                              "http://default.url",
		Password:                         "default_password",
		Username:                         "default_username",
		SubaccountWithRequestCredentials: "special_subaccountID",
	})

	for tn, tc := range map[string]struct {
		givenRequest        RequestContext
		expectedCredentials *Credentials
	}{
		"regular SubAccount": {
			givenRequest: RequestContext{
				SubaccountID:           "regular_saID",
				Credentials:            nil,
				BTPOperatorCredentials: nil,
			},
			expectedCredentials: &Credentials{
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
			expectedCredentials: &Credentials{
				Password: "p",
				Username: "u",
				URL:      "http://url",
			},
		},
		"with BTP Operator credentials": {
			givenRequest: RequestContext{
				SubaccountID: "regular_saID",
				Credentials:  nil,
				BTPOperatorCredentials: &BTPOperatorCredentials{
					ClientID:     "c-id",
					ClientSecret: "c-s",
					TokenURL:     "http://btp-url",
					ClusterID:    "cl-id",
				},
			},
			expectedCredentials: nil,
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
