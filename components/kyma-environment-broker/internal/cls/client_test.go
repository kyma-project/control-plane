package cls

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type provisioningInputMatcher func(input servicemanager.ProvisioningInput) bool

var config = &Config{
	RetentionPeriod:    30,
	MaxDataInstances:   4,
	MaxIngestInstances: 4,
	SAML: &SAMLConfig{
		AdminGroup:  "runtimeAdmin",
		ExchangeKey: "base64-jibber-jabber",
		Initiated:   true,
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

func TestCreateInstance(t *testing.T) {
	const (
		fakeBrokerID   = "fake-broker-id"
		fakeServiceID  = "fake-service-id"
		fakePlanID     = "fake-plan-id"
		fakeInstanceID = "fake-instance-id"
	)

	tests := []struct {
		summary string
		matcher provisioningInputMatcher
	}{
		{
			"service id is set",
			func(input servicemanager.ProvisioningInput) bool {
				return input.ServiceID == fakeServiceID
			},
		},
		{
			"plan id is set",
			func(input servicemanager.ProvisioningInput) bool {
				return input.PlanID == fakePlanID
			},
		},
		{
			"instance id is set",
			func(input servicemanager.ProvisioningInput) bool {
				return input.ID == fakeInstanceID
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
		{
			"platform is kubernetes",
			func(input servicemanager.ProvisioningInput) bool {
				if platform, ok := input.Context["platform"]; ok {
					return platform == "kubernetes"
				}
				return false
			},
		},
		{
			"elk parameters are set",
			func(input servicemanager.ProvisioningInput) bool {
				params := input.Parameters.(parameters)
				return params.RetentionPeriod == config.RetentionPeriod &&
					params.MaxDataInstances == config.MaxDataInstances &&
					params.MaxIngestInstances == config.MaxIngestInstances &&
					params.EsAPIEnabled == false
			},
		},
		{
			"saml parameters are set",
			func(input servicemanager.ProvisioningInput) bool {
				params := input.Parameters.(parameters)
				return params.SAML.Enabled == true &&
					params.SAML.Initiated == config.SAML.Initiated &&
					params.SAML.AdminGroup == config.SAML.AdminGroup &&
					params.SAML.ExchangeKey == config.SAML.ExchangeKey &&
					params.SAML.RolesKey == config.SAML.RolesKey &&
					params.SAML.Idp.EntityID == config.SAML.Idp.EntityID &&
					params.SAML.Idp.MetadataURL == config.SAML.Idp.MetadataURL &&
					params.SAML.Sp.EntityID == config.SAML.Sp.EntityID &&
					params.SAML.Sp.SignaturePrivateKey == config.SAML.Sp.SignaturePrivateKey
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			// given
			smClientMock := &automock.Client{}
			smClientMock.On("Provision", fakeBrokerID, mock.MatchedBy(tc.matcher), true).Return(&servicemanager.ProvisionResponse{}, nil)
			sut := NewClient(config, logger.NewLogDummy())

			// when
			err := sut.CreateInstance(smClientMock, servicemanager.InstanceKey{
				BrokerID:   fakeBrokerID,
				ServiceID:  fakeServiceID,
				PlanID:     fakePlanID,
				InstanceID: fakeInstanceID,
			})

			// then
			require.NoError(t, err)
		})
	}
}

func TestCreateBinding(t *testing.T) {
	smClientMock := &automock.Client{}
	creds := make(map[string]interface{})
	creds["Kibana-endpoint"] = "kibUrl"
	creds["Fluentd-username"] = "fbUser"
	creds["Fluentd-password"] = "fbPass"
	creds["Fluentd-endpoint"] = "fbEndpoint"
	resB := servicemanager.BindingResponse{
		Binding:      servicemanager.Binding{Credentials: creds},
		HTTPResponse: servicemanager.HTTPResponse{StatusCode: 200},
	}
	smClientMock.On("Bind", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&resB, nil)
	sut := NewClient(config, logrus.New())

	br := BindingRequest{
		InstanceKey: servicemanager.InstanceKey{},
		BindingID:   "",
	}
	expectedClsOverrides := &OverrideParams{
		FluentdEndPoint: "fbEndpoint",
		FluentdPassword: "fbPass",
		FluentdUsername: "fbUser",
		KibanaUrl:       "kibUrl",
	}
	res, err := sut.CreateBinding(smClientMock, &br)
	require.NoError(t, err)
	require.Equal(t, expectedClsOverrides, res)
}

func TestRemoveInstance(t *testing.T) {
	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	tests := []struct {
		summary        string
		deprovisionErr error
		expectedErr    bool
	}{
		{
			summary:        "deprovision fails",
			deprovisionErr: errors.New("unable to connect"),
			expectedErr:    true,
		},
		{
			summary:     "deprovision succeeds",
			expectedErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.summary, func(t *testing.T) {
			// given
			smClientMock := &automock.Client{}
			smClientMock.On("Deprovision", fakeInstance, true).Return(&servicemanager.DeprovisionResponse{}, tc.deprovisionErr)
			sut := NewClient(new(Config), logger.NewLogDummy())

			// when
			err := sut.RemoveInstance(smClientMock, fakeInstance)

			// then
			if tc.expectedErr {
				require.Error(t, err)
			}
		})
	}

}
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
