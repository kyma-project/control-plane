package deprovisioning

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	edpName        = "f88401ba-c601-45bb-bec0-a2156c07c9a6"
	edpEnvironment = "test"
)

func TestEDPDeregistration_Run(t *testing.T) {
	// given
	client := edp.NewFakeClient()
	err := client.CreateDataTenant(edp.DataTenantPayload{
		Name:        edpName,
		Environment: edpEnvironment,
		Secret:      base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", edpName, edpEnvironment))),
	})
	assert.NoError(t, err)

	metadataTenantKeys := []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	}

	for _, key := range metadataTenantKeys {
		err = client.CreateMetadataTenant(edpName, edpEnvironment, edp.MetadataTenantPayload{
			Key:   key,
			Value: "-",
		})
		assert.NoError(t, err)
	}

	step := NewEDPDeregistrationStep(client, edp.Config{
		Environment: edpEnvironment,
	})

	// when
	_, repeat, err := step.Run(internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				SubAccountID: edpName,
			},
		}}, logrus.New())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		metadataTenant, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.False(t, metadataTenantExists)
		assert.Equal(t, edp.MetadataItem{}, metadataTenant)
	}

	dataTenant, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.False(t, dataTenantExists)
	assert.Equal(t, edp.DataTenantItem{}, dataTenant)
}
