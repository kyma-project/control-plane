package cls

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	fakeBrokerID = "fake-broker-id"
)

var (
	config = &Config{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		SAML: &SAMLConfig{
			AdminGroup:  "runtimeAdmin",
			ExchangeKey: "base64-jibber-jabber",
			RolesKey:    "groups",
			Idp: &SAMLIdpConfig{
				EntityID:    "https://sso.example.org/idp",
				MetadataURL: "https://sso.example.org/idp/saml2/metadata",
			},
			Sp: &SAMLSpConfig{
				EntityID:            "cls-dev",
				SignaturePrivateKey: "base64-jibber-jabber",
			},
		},
	}
)

type provisioningInputMatcher func(input servicemanager.ProvisioningInput) bool

func TestCreateInstance(t *testing.T) {
	tests := []struct {
		summary string
		matcher provisioningInputMatcher
	}{
		{
			"instance id is valid uuid",
			func(input servicemanager.ProvisioningInput) bool {
				return isValidUUID(input.ID)
			},
		},
		{
			"organization id is valid uuid",
			func(input servicemanager.ProvisioningInput) bool {
				return isValidUUID(input.OrganizationGUID)
			},
		},
		{
			"space id is valid uuid",
			func(input servicemanager.ProvisioningInput) bool {
				return isValidUUID(input.SpaceGUID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			smClientMock := &automock.Client{}
			smClientMock.On("Provision", fakeBrokerID, mock.MatchedBy(tc.matcher), true).Return(&servicemanager.ProvisionResponse{}, nil)
			sut := NewClient(config, logrus.New())

			instanceID, err := sut.CreateInstance(smClientMock, &CreateInstanceRequest{
				BrokerID: fakeBrokerID,
			})
			require.NotNil(t, instanceID)
			require.NoError(t, err)
		})
	}
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
