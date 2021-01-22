package deprovisioning

import (
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
}
