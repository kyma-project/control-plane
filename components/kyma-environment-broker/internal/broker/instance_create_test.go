package broker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	serviceID       = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	planID          = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	globalAccountID = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	subAccountID    = "3cb65e5b-e455-4799-bf35-be46e8f5a533"

	instanceID       = "d3d5dca4-5dc8-44ee-a825-755c2a3fb839"
	otherInstanceID  = "87bfaeaa-48eb-40d6-84f3-3d5368eed3eb\n"
	existOperationID = "920cbfd9-24e9-4aa2-aa77-879e9aabe140"
	clusterName      = "cluster-testing"
	region           = "eu"
)

func TestProvision_Provision(t *testing.T) {
	t.Run("new operation will be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},

			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, globalAccountID, subAccountID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)
		assert.Regexp(t, `^https:\/\/console\.[a-z0-9\-]{7,9}\.test\.example\.com`, response.DashboardURL)
		assert.Equal(t, clusterName, response.Metadata.Labels["Name"])
		assert.Regexp(t, `^https:\/\/grafana\.[a-z0-9\-]{7,9}\.test\.example\.com`, response.Metadata.Labels["GrafanaURL"])

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Regexp(t, `^https:\/\/console\.[a-z0-9\-]{7,9}\.test\.example\.com`, instance.DashboardURL)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
	})

	t.Run("existing operation ID will be return", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixExistOperation())
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			nil,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, region), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, globalAccountID, subAccountID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Equal(t, existOperationID, response.OperationData)
		assert.True(t, response.AlreadyExists)
	})

	t.Run("more than one trial is not allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: globalAccountID,
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			nil,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		_, err = provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), "new-instance-id", domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, globalAccountID, subAccountID)),
		}, true)

		// then
		assert.EqualError(t, err, "The Trial Kyma was created for the global account, but there is only one allowed")
	})

	t.Run("more than one trial is allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: globalAccountID,
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		assert.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: false},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "req-region"), otherInstanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, globalAccountID, subAccountID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, otherInstanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		instance, err := memoryStorage.Instances().GetByID(otherInstanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
	})

	t.Run("provision trial", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: "other-global-account",
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, globalAccountID, subAccountID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
	})

	t.Run("conflict should be handled", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(fixInstance())
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			nil,
			factoryBuilder,
			fixAlwaysPassJSONValidator(),
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)

		// then
		assert.EqualError(t, err, "provisioning operation already exist")
		assert.Empty(t, response.OperationData)
	})

	t.Run("kyma version parameters should be saved", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		fixValidator, err := broker.NewPlansSchemaValidator(broker.PlansConfig{})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixValidator,
			broker.PlansConfig{},
			true,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID: serviceID,
			PlanID:    planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{
								"name": "%s",
								"kymaVersion": "master-00e83e99"
								}`, clusterName)),
			RawContext: json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, "master-00e83e99", operation.ProvisioningParameters.Parameters.KymaVersion)
	})

	t.Run("should return error when region is not specified", func(t *testing.T) {
		// given
		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		fixValidator, err := broker.NewPlansSchemaValidator(broker.PlansConfig{})
		require.NoError(t, err)

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			nil,
			nil,
			nil,
			factoryBuilder,
			fixValidator,
			broker.PlansConfig{},
			true,
			logrus.StandardLogger(),
		)

		// when
		_, provisionErr := provisionEndpoint.Provision(context.Background(), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)

		// then
		require.EqualError(t, provisionErr, "No region specified in request.")
	})

	t.Run("kyma version parameters should NOT be saved", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		fixValidator, err := broker.NewPlansSchemaValidator(broker.PlansConfig{})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixValidator,
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID: serviceID,
			PlanID:    planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{
								"name": "%s",
								"kymaVersion": "master-00e83e99"
								}`, clusterName)),
			RawContext: json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, "", operation.ProvisioningParameters.Parameters.KymaVersion)
	})

	t.Run("licence type lite should be saved in parameters", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AzureLitePlanID).Return(true)

		fixValidator, err := broker.NewPlansSchemaValidator(broker.PlansConfig{})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixValidator,
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AzureLitePlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, ptr.String(internal.LicenceTypeLite), operation.ProvisioningParameters.Parameters.LicenceType)
	})

	t.Run("licence type lite should be saved in parameters for Trial Plan", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		fixValidator, err := broker.NewPlansSchemaValidator(broker.PlansConfig{})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com"},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			queue,
			factoryBuilder,
			fixValidator,
			broker.PlansConfig{},
			false,
			logrus.StandardLogger(),
		)

		// when
		response, err := provisionEndpoint.Provision(fixReqCtxWithRegion(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, ptr.String(internal.LicenceTypeLite), operation.ProvisioningParameters.Parameters.LicenceType)
	})
}

func fixExistOperation() internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(existOperationID, instanceID)
	pp := internal.ProvisioningParameters{}
	pp.PlanID = planID
	pp.ServiceID = serviceID
	pp.ErsContext.SubAccountID = subAccountID
	pp.ErsContext.GlobalAccountID = globalAccountID
	pp.Parameters.Name = clusterName
	pp.PlatformRegion = region
	provisioningOperation.ProvisioningParameters = pp

	return provisioningOperation
}

func fixAlwaysPassJSONValidator() broker.PlansSchemaValidator {
	validatorMock := &automock.JSONSchemaValidator{}
	validatorMock.On("ValidateString", mock.Anything).Return(jsonschema.ValidationResult{Valid: true}, nil)

	fixValidator := broker.PlansSchemaValidator{
		broker.GCPPlanID:   validatorMock,
		broker.AzurePlanID: validatorMock,
		broker.TrialPlanID: validatorMock,
	}

	return fixValidator
}

func fixInstance() internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.GlobalAccountID = globalAccountID
	instance.ServiceID = serviceID
	instance.ServicePlanID = planID

	return instance
}

func fixReqCtxWithRegion(t *testing.T, region string) context.Context {
	t.Helper()

	req, err := http.NewRequest("GET", "http://url.io", nil)
	require.NoError(t, err)
	var ctx context.Context
	spyHandler := http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		ctx = req.Context()
	})

	middleware.AddRegionToContext(region).Middleware(spyHandler).ServeHTTP(httptest.NewRecorder(), req)
	return ctx
}
