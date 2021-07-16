package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBusolaMigratorOverridesStep_Run(t *testing.T) {
	t.Run("testing BusolaMigratorOverrides", func(t *testing.T) {
		// given
		givenOperation := fixture.FixProvisioningOperation("7a5ab267-7826-4208-acd2-f5bbcf6de966", "2d05a736-09f4-40de-80eb-37c6d5fc91ca")
		ic := newInputCreator()
		givenOperation.InputCreator = ic

		step := NewBusolaMigratorOverridesStep()

		// when
		_, repeat, err := step.Run(givenOperation, logrus.New())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		ic.AssertOverride(t, BusolaMigratorComponentName, gqlschema.ConfigEntryInput{
			Key:   "deployment.env.kubeconfigID",
			Value: givenOperation.InstanceID,
		})
	})
}
