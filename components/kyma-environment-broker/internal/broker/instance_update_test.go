package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

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

type handler struct {
	Instance   internal.Instance
	ersContext internal.ERSContext
}

func (h *handler) Handle(inst *internal.Instance, ers internal.ERSContext) error {
	h.Instance = *inst
	h.ersContext = ers
	return nil
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
	svc := NewUpdate(Config{}, st.Instances(), st.Operations(), handler, true, &q, logrus.New())

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
	svc := NewUpdate(Config{}, st.Instances(), st.Operations(), handler, true, q, logrus.New())

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
	svc := NewUpdate(Config{}, st.Instances(), st.Operations(), handler, true, q, logrus.New())

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
	svc := NewUpdate(Config{}, st.Instances(), st.Operations(), handler, true, q, logrus.New())

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
