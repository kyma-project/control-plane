package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
)

const (
	edpName        = "cd4b333c-97fb-4894-bb20-7874f5833e8d"
	edpEnvironment = "test"
	edpRegion      = "cf-eu10"
	edpPlan        = "standard"
)

func TestEDPRegistration_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	client := edp.NewFakeClient()

	step := NewEDPRegistrationStep(memoryStorage.Operations(), client, edp.Config{
		Environment: edpEnvironment,
		Required:    true,
	})
	operation := internal.Operation{
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID:         broker.AzurePlanID,
			PlatformRegion: edpRegion,
			ErsContext: internal.ERSContext{
				SubAccountID: edpName,
			},
		},
	}
	memoryStorage.Operations().InsertOperation(operation)

	// when
	_, repeat, err := step.Run(operation, logger.NewLogDummy())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	dataTenant, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.True(t, dataTenantExists)
	assert.Equal(t, edp.DataTenantItem{
		Name:        edpName,
		Environment: edpEnvironment,
	}, dataTenant)

	for key, value := range map[string]string{
		edp.MaasConsumerEnvironmentKey: step.selectEnvironmentKey(edpRegion, logger.NewLogDummy()),
		edp.MaasConsumerRegionKey:      edpRegion,
		edp.MaasConsumerSubAccountKey:  edpName,
		edp.MaasConsumerServicePlan:    edpPlan,
	} {
		metadataTenant, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.True(t, metadataTenantExists)
		assert.Equal(t, edp.MetadataItem{
			DataTenant: edp.DataTenantItem{
				Name:        edpName,
				Environment: edpEnvironment,
			},
			Key:   key,
			Value: value,
		}, metadataTenant)
	}

}

func TestEDPRegistrationStep_selectEnvironmentKey(t *testing.T) {
	for name, tc := range map[string]struct {
		region   string
		expected string
	}{
		"kubernetes region": {
			region:   "k8s-as34",
			expected: "KUBERNETES",
		},
		"cf region": {
			region:   "cf-eu10",
			expected: "CF",
		},
		"neo region": {
			region:   "neo-us13",
			expected: "NEO",
		},
		"default region": {
			region:   "undefined",
			expected: "CF",
		},
		"empty region": {
			region:   "",
			expected: "CF",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			step := NewEDPRegistrationStep(nil, nil, edp.Config{})

			// when
			envKey := step.selectEnvironmentKey(tc.region, logger.NewLogDummy())

			// then
			assert.Equal(t, tc.expected, envKey)
		})
	}
}

func TestEDPRegistrationStep_selectServicePlan(t *testing.T) {
	for name, tc := range map[string]struct {
		planID   string
		expected string
	}{
		"GCP": {
			planID:   broker.GCPPlanID,
			expected: "standard",
		},
		"AWS": {
			planID:   broker.AWSPlanID,
			expected: "standard",
		},
		"Azure": {
			planID:   broker.AzurePlanID,
			expected: "standard",
		},
		"Azure Lite": {
			planID:   broker.AzureLitePlanID,
			expected: "tdd",
		},
		"Trial": {
			planID:   broker.TrialPlanID,
			expected: "standard",
		},
		"OpenStack": {
			planID:   broker.OpenStackPlanID,
			expected: "standard",
		},
		"Freemium": {
			planID:   broker.FreemiumPlanID,
			expected: "free",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			step := NewEDPRegistrationStep(nil, nil, edp.Config{})

			// when
			envKey := step.selectServicePlan(tc.planID)

			// then
			assert.Equal(t, tc.expected, envKey)
		})
	}
}
