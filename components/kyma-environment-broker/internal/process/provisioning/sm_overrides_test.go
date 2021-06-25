package provisioning

import (
	"io/ioutil"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate mockery -name=ProvisionerInputCreator -dir=../../ -output=automock -outpkg=automock -case=underscore

func TestServiceManagerOverridesStepSuccess(t *testing.T) {
	ts := SMOverrideTestSuite{}

	tests := map[string]struct {
		requestParams        internal.ProvisioningParameters
		overrideParams       servicemanager.Config
		expCredentialsValues []*gqlschema.ConfigEntryInput
	}{
		"always apply override for Service Manager credentials": {
			requestParams:  ts.SMRequestParameters("req-url", "req-user", "req-pass"),
			overrideParams: ts.SMOverrideConfig(servicemanager.SMOverrideModeAlways, "over-url", "over-user", "over-pass"),

			expCredentialsValues: []*gqlschema.ConfigEntryInput{
				{Key: "config.sm.url", Value: "over-url"},
				{Key: "sm.user", Value: "over-user"},
				{Key: "sm.password", Value: "over-pass", Secret: ptr.Bool(true)},
			},
		},
		"never apply override for Service Manager credentials": {
			requestParams:  ts.SMRequestParameters("req-url", "req-user", "req-pass"),
			overrideParams: ts.SMOverrideConfig(servicemanager.SMOverrideModeNever, "over-url", "over-user", "over-pass"),

			expCredentialsValues: []*gqlschema.ConfigEntryInput{
				{Key: "config.sm.url", Value: "req-url"},
				{Key: "sm.user", Value: "req-user"},
				{Key: "sm.password", Value: "req-pass", Secret: ptr.Bool(true)},
			},
		},
		"apply override for Service Manager credentials because they are not present in request": {
			requestParams:  internal.ProvisioningParameters{},
			overrideParams: ts.SMOverrideConfig(servicemanager.SMOverrideModeWhenNotSentInRequest, "over-url", "over-user", "over-pass"),

			expCredentialsValues: []*gqlschema.ConfigEntryInput{
				{Key: "config.sm.url", Value: "over-url"},
				{Key: "sm.user", Value: "over-user"},
				{Key: "sm.password", Value: "over-pass", Secret: ptr.Bool(true)},
			},
		},
		"do not apply override for Service Manager credentials because they are present in request": {
			requestParams:  ts.SMRequestParameters("req-url", "req-user", "req-pass"),
			overrideParams: ts.SMOverrideConfig(servicemanager.SMOverrideModeWhenNotSentInRequest, "over-url", "over-user", "over-pass"),

			expCredentialsValues: []*gqlschema.ConfigEntryInput{
				{Key: "config.sm.url", Value: "req-url"},
				{Key: "sm.user", Value: "req-user"},
				{Key: "sm.password", Value: "req-pass", Secret: ptr.Bool(true)},
			},
		},
	}
	for tN, tC := range tests {
		t.Run(tN, func(t *testing.T) {
			// given
			inputCreatorMock := &automock.ProvisionerInputCreator{}
			inputCreatorMock.On("AppendOverrides", "service-manager-proxy", tC.expCredentialsValues).
				Return(nil).Once()

			factory := servicemanager.NewClientFactory(tC.overrideParams)
			operation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ProvisioningParameters: tC.requestParams,
				},
				InputCreator:    inputCreatorMock,
				SMClientFactory: factory,
			}

			memoryStorage := storage.NewMemoryStorage()
			smStep := NewServiceManagerOverridesStep(memoryStorage.Operations())

			// when
			gotOperation, retryTime, err := smStep.Run(operation, NewLogDummy())

			// then
			require.NoError(t, err)

			assert.Zero(t, retryTime)
			assert.Equal(t, operation, gotOperation)
			inputCreatorMock.AssertExpectations(t)
		})
	}
}

func TestServiceManagerOverridesStepError(t *testing.T) {
	tests := map[string]struct {
		givenReqParams internal.ProvisioningParameters
		expErr         string
	}{
		"return error when creds in request are not provided and overrides should not be applied": {
			givenReqParams: internal.ProvisioningParameters{},
			expErr:         "Service Manager Credentials are required to be send in provisioning request.",
		},
	}
	for tN, tC := range tests {
		t.Run(tN, func(t *testing.T) {
			// given
			factory := servicemanager.NewClientFactory(servicemanager.Config{
				OverrideMode: servicemanager.SMOverrideModeNever,
				URL:          "",
				Password:     "",
				Username:     "",
			})
			operation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:                     "123",
					ProvisioningParameters: tC.givenReqParams,
				},
				SMClientFactory: factory,
			}

			memoryStorage := storage.NewMemoryStorage()
			require.NoError(t, memoryStorage.Operations().InsertProvisioningOperation(operation))
			smStep := NewServiceManagerOverridesStep(memoryStorage.Operations())

			// when
			gotOperation, retryTime, err := smStep.Run(operation, NewLogDummy())

			// then
			require.EqualError(t, err, tC.expErr)
			assert.Zero(t, retryTime)
			assert.Equal(t, domain.Failed, gotOperation.State)
		})
	}
}

type SMOverrideTestSuite struct{}

func (SMOverrideTestSuite) SMRequestParameters(smURL, smUser, smPass string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			ServiceManager: &internal.ServiceManagerEntryDTO{URL: smURL,
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: smUser,
						Password: smPass,
					}}},
		},
	}
}

func (s SMOverrideTestSuite) SMOverrideConfig(mode servicemanager.ServiceManagerOverrideMode, url string, user string, pass string) servicemanager.Config {
	return servicemanager.Config{
		OverrideMode: mode,
		URL:          url,
		Username:     user,
		Password:     pass,
	}
}

// NewLogDummy returns dummy logger which discards logged messages on the fly.
// Useful when logger is required as dependency in unit testing.
func NewLogDummy() *logrus.Entry {
	rawLgr := logrus.New()
	rawLgr.Out = ioutil.Discard
	lgr := rawLgr.WithField("testing", true)

	return lgr
}
