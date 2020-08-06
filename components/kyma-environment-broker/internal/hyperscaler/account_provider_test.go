package hyperscaler

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultTenant = "default-tenant"

func TestGardenerSecretNamePreAssigned(t *testing.T) {

	pool := newTestAccountPool()

	accountProvider := NewAccountProvider(nil, pool, nil)

	configInput := &gqlschema.GardenerConfigInput{
		TargetSecret: "pre-assigned-secret",
	}

	secretName, err := accountProvider.GardenerSecretName(configInput, defaultTenant)
	require.NoError(t, err)

	assert.Equal(t, "pre-assigned-secret", secretName)
}

func TestGardenerSecretNamePool(t *testing.T) {

	pool := newTestAccountPool()

	accountProvider := NewAccountProvider(nil, pool, nil)

	configInput := &gqlschema.GardenerConfigInput{
		Provider:     "AWS",
		TargetSecret: "",
	}

	secretName, err := accountProvider.GardenerSecretName(configInput, defaultTenant)
	require.NoError(t, err)

	assert.Equal(t, "secret5", secretName)
}

func TestGardenerSecretNameError(t *testing.T) {

	pool := newTestAccountPool()

	accountProvider := NewAccountProvider(nil, pool, nil)

	configInput := &gqlschema.GardenerConfigInput{
		Provider:     "bogus",
		TargetSecret: "",
	}

	_, err := accountProvider.GardenerSecretName(configInput, defaultTenant)

	require.Error(t, err)

	assert.Equal(t, "unknown Hyperscaler provider type: bogus", err.Error())
}

func TestGardenerSecretNameNotFound(t *testing.T) {

	pool := newTestAccountPool()

	accountProvider := NewAccountProvider(nil, pool, nil)

	configInput := &gqlschema.GardenerConfigInput{
		Provider:     "azure",
		TargetSecret: "",
	}

	_, err := accountProvider.GardenerSecretName(configInput, defaultTenant)

	require.Error(t, err)

	assert.Equal(t, "accountPool failed to find unassigned secret for hyperscalerType: azure", err.Error())
}

func TestHyperscalerTypeFromProvisionInputGardenerGCP(t *testing.T) {

	input := &gqlschema.ProvisionRuntimeInput{
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Provider: "GCP",
			},
		},
	}

	hyperscalerType, err := HyperscalerTypeFromProvisionInput(input)
	require.NoError(t, err)

	assert.Equal(t, hyperscalerType, GCP)
}

func TestHyperscalerTypeFromProvisionInputGardenerAWS(t *testing.T) {

	input := &gqlschema.ProvisionRuntimeInput{
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Provider: "AWS",
			},
		},
	}

	hyperscalerType, err := HyperscalerTypeFromProvisionInput(input)
	require.NoError(t, err)

	assert.Equal(t, hyperscalerType, AWS)
}

func TestHyperscalerTypeFromProvisionInputGardenerAZURE(t *testing.T) {

	input := &gqlschema.ProvisionRuntimeInput{
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Provider: "AZURE",
			},
		},
	}

	hyperscalerType, err := HyperscalerTypeFromProvisionInput(input)
	require.NoError(t, err)

	assert.Equal(t, hyperscalerType, Azure)
}

func TestHyperscalerTypeFromProvisionInputGardenerError(t *testing.T) {

	input := &gqlschema.ProvisionRuntimeInput{
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Provider: "bogus",
			},
		},
	}

	hyperscalerType, err := HyperscalerTypeFromProvisionInput(input)

	require.Error(t, err)
	assert.Empty(t, hyperscalerType)
	assert.Equal(t, "unknown Hyperscaler provider type: bogus", err.Error())
}

func TestHyperscalerTypeFromProvisionInputError(t *testing.T) {

	_, err := HyperscalerTypeFromProvisionInput(nil)

	require.Error(t, err)

	assert.Equal(t, err.Error(), "can't determine hyperscaler type because ProvisionRuntimeInput not specified (was nil)")

	input := &gqlschema.ProvisionRuntimeInput{}

	_, err = HyperscalerTypeFromProvisionInput(input)

	require.Error(t, err)

	assert.Equal(t, err.Error(), "can't determine hyperscaler type because ProvisionRuntimeInput.ClusterConfig not specified (was nil)")

	input = &gqlschema.ProvisionRuntimeInput{
		ClusterConfig: &gqlschema.ClusterConfigInput{},
	}

	_, err = HyperscalerTypeFromProvisionInput(input)

	require.Error(t, err)

	assert.Equal(t, err.Error(), "can't determine hyperscaler type because ProvisionRuntimeInput.ClusterConfig.GardenerConfig not specified (was nil)")
}

func TestGardenerSharedCredentials_Error(t *testing.T) {

	accountProvider := NewAccountProvider(nil, nil, nil)

	_, err := accountProvider.GardenerSharedCredentials(Type("gcp"))
	require.Error(t, err)

	assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
}

func TestReleaseGardenerSecretForLastCluster(t *testing.T) {

	pool := newTestAccountPoolWithSingleShoot()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azure"), "tenant1")

	require.NoError(t, err)
}

func TestReleaseGardenerSecretForLastCluster_AlreadyReleased(t *testing.T) {

	pool := newTestAccountPoolWithSingleShootReleased()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azure"), "tenant1")

	require.NoError(t, err)
}

func TestReleaseGardenerSecretForMultipleClusters(t *testing.T) {

	pool := newTestAccountPoolWithMultipleShoots()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azure"), "tenant1")

	require.NoError(t, err)
}

func TestReleaseGardenerSecretForMultipleClusters_ErrorNoPool(t *testing.T) {

	accountProvider := NewAccountProvider(nil, nil, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("gcp"), "tenant1")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "failed to release subscription for tenant. Gardener Account pool is not configured")
}

func TestReleaseGardenerSecret_ErrorBadTenant(t *testing.T) {

	pool := newTestAccountPoolWithSingleShoot()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azure"), "tenantX")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "accountPool failed to find subscription secret used by the tenant tenantX and hyperscaler azure")
}

func TestReleaseGardenerSecret_ErrorBadHyperscaler(t *testing.T) {

	pool := newTestAccountPoolWithSingleShoot()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azureX"), "tenant1")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "accountPool failed to find subscription secret used by the tenant tenant1 and hyperscaler azureX")
}

// stange case but it should pass anyway
func TestReleaseGardenerSecret_NoShoots(t *testing.T) {

	pool := newTestAccountPoolNoValidShoots()

	accountProvider := NewAccountProvider(nil, pool, nil)

	err := accountProvider.ReleaseGardenerSecretForLastCluster(Type("azure"), "tenant1")

	require.NoError(t, err)
}

//func newShoot(name, secret string) *gardener_types.Shoot {
//	return &gardener_types.Shoot{
//		ObjectMeta: machineryv1.ObjectMeta{
//			Name:      name,
//			Namespace: testNamespace,
//		},
//		Spec: gardener_types.ShootSpec{
//			SecretBindingName: secret,
//		},
//	}
//}
