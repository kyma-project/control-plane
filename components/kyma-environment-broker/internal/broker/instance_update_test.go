package broker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enabledDashboardConfig dashboard.Config = dashboard.Config{Enabled: true, LandscapeURL: "https://dashboard.example.com"}

type handler struct {
	Instance   internal.Instance
	ersContext internal.ERSContext
}

func (h *handler) Handle(inst *internal.Instance, ers internal.ERSContext) (bool, error) {
	h.Instance = *inst
	h.ersContext = ers
	return false, nil
}

func TestUpdateEndpoint_UpdateSuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))

	handler := &handler{}
	q := process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, &q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)
	// check if original ERS context is set again in the Instance entity
	assert.NotEmpty(t, inst.Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Password)
	// check if the handler was called
	assertServiceManagerCreds(t, handler.Instance.Parameters.ErsContext.ServiceManager)

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)
}

func TestUpdateEndpoint_UpdateUnsuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())

	handler := &handler{}
	q := &process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)
	// check if original ERS context is set again in the Instance entity
	assert.NotEmpty(t, inst.Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Password)
	// check if the handler was called
	assertServiceManagerCreds(t, handler.Instance.Parameters.ErsContext.ServiceManager)

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(true),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.False(t, *handler.Instance.Parameters.ErsContext.Active)
}

func assertServiceManagerCreds(t *testing.T, dto *internal.ServiceManagerEntryDTO) {
	assert.Equal(t, &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: "u",
				Password: "p",
			},
		}}, dto)
}

func TestUpdateEndpoint_UpdateInstanceWithWrongActiveValue(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          ptr.Bool(false),
			},
		},
	}
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	handler := &handler{}
	q := &process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	inst, _ := st.Instances().GetByID(instanceID)
	// check if original ERS context is set again in the Instance entity
	assert.NotEmpty(t, inst.Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Password)
	// check if the handler was called
	assertServiceManagerCreds(t, handler.Instance.Parameters.ErsContext.ServiceManager)
	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	handler := &handler{}
	q := &process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusNotFound, "Updating status code not matching")
}

func fixProvisioningOperation(id string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(id, instanceID)
	provisioningOperation.ProvisioningParameters.ErsContext.ServiceManager.URL = ""

	return provisioningOperation
}

func fixSuspensionOperation() internal.DeprovisioningOperation {
	deprovisioningOperation := fixture.FixDeprovisioningOperation("id", instanceID)
	deprovisioningOperation.ProvisioningParameters.ErsContext.ServiceManager.URL = ""
	deprovisioningOperation.Temporary = true

	return deprovisioningOperation
}

func TestUpdateEndpoint_UpdateGlobalAccountID(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "origin-account-id",
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          nil,
			},
		},
	}
	newGlobalAccountID := "updated-account-id"
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))

	handler := &handler{}
	q := process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, &q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"" + newGlobalAccountID + "\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)
	// Check if SubscriptionGlobalAccountID is not empty
	assert.NotEmpty(t, inst.SubscriptionGlobalAccountID)
	// check if the handler was called
	assertServiceManagerCreds(t, handler.Instance.Parameters.ErsContext.ServiceManager)

	// Check if SubscriptionGlobalAccountID is now the same as GlobalAccountID
	assert.Equal(t, inst.GlobalAccountID, newGlobalAccountID)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)
}

func TestUpdateEndpoint_UpdateParameters(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))

	handler := &handler{}
	q := process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, &q, planDefaults, logrus.New(), enabledDashboardConfig)

	t.Run("Should fail on invalid OIDC params", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"{clientID}","groupsClaim":"groups","issuerURL":"{issuerURL}","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-"`
		errMsg := errors.New("issuerURL must be a valid URL, issuerURL must have https scheme")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on insufficient OIDC params (missing issuerURL)", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"client-id"`
		errMsg := errors.New("issuerURL must not be empty")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on insufficient OIDC params (missing clientID)", func(t *testing.T) {
		// given
		oidcParams := `"issuerURL":"https://test.local"`
		errMsg := errors.New("clientID must not be empty")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC signingAlgs param", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256","notValid"]`
		errMsg := errors.New("signingAlgs must contain valid signing algorithm(s)")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})
}

func TestUpdateEndpoint_UpdateWithEnabledDashboard(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          nil,
			},
		},
		DashboardURL: "https://console.cd6e47b.example.com",
	}
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	// st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	// st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))

	handler := &handler{}
	q := process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, &q, planDefaults, logrus.New(), enabledDashboardConfig)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)

	// check if the instance is updated successfully
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, inst.DashboardURL)
	// check if the API response is correct
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
}

func TestUpdateEndpoint_UpdateWithDisabledDashboard(t *testing.T) {
	// given
	disabledDashboardConfig := dashboard.Config{
		Enabled:      false,
		LandscapeURL: "example.com",
	}

	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				ServiceManager:  nil,
				Active:          nil,
			},
		},
		DashboardURL: "https://console.cd6e47b.example.com",
	}
	st := storage.NewMemoryStorage()
	st.Instances().Insert(instance)
	st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))

	handler := &handler{}
	q := process.Queue{}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, &q, planDefaults, logrus.New(), disabledDashboardConfig)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)

	// ensure the instance is not updated
	assert.Regexp(t, `^https:\/\/console\.[a-z0-9\-]{7,9}\.example\.com`, inst.DashboardURL)
	// ensure the API response is not updated
	assert.Regexp(t, `^https:\/\/console\.[a-z0-9\-]{7,9}\.example\.com`, response.DashboardURL)
}
