package broker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetEndpoint_GetNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	svc := broker.NewGetInstance(broker.Config{}, st.Instances(), st.Operations(), logrus.New())

	// when
	_, err := svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, http.StatusNotFound, apierr.ValidatedStatusCode(nil), "Get status code not matching")
}

func TestGetEndpoint_GetProvisioningInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", planID).Return(true)

	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	createSvc := broker.NewProvision(
		broker.Config{EnablePlans: []string{"gcp", "azure", "azure_ha"}, OnlySingleTrialPerGA: true},
		gardener.Config{Project: "test", ShootDomain: "example.com"},
		st.Operations(),
		st.Instances(),
		queue,
		factoryBuilder,
		broker.PlansConfig{},
		false,
		planDefaults,
		logrus.StandardLogger(),
		kubeconfigOrigin,
	)
	getSvc := broker.NewGetInstance(broker.Config{EnableKubeconfigURLLabel: true}, st.Instances(), st.Operations(), logrus.New())

	// when
	createSvc.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
		ServiceID:     serviceID,
		PlanID:        planID,
		RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
		RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
	}, true)

	// then
	_, err := getSvc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, http.StatusNotFound, apierr.ValidatedStatusCode(nil), "Get status code not matching")
	assert.Equal(t, "provisioning of instanceID d3d5dca4-5dc8-44ee-a825-755c2a3fb839 in progress", apierr.Error())

	// when
	op, _ := st.Operations().GetProvisioningOperationByInstanceID(instanceID)
	op.State = domain.Succeeded
	st.Operations().UpdateProvisioningOperation(*op)

	// then
	response, err := getSvc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})
	assert.Equal(t, nil, err, "Get returned error when expected to pass")
	assert.Len(t, response.Metadata.Labels, 2)
}
