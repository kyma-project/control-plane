package upgrade_kyma

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
		wantOp := fixture.FixUpgradeKymaOperation("7a5ab267-7826-4208-acd2-f5bbcf6de966", "2d05a736-09f4-40de-80eb-37c6d5fc91ca")

		wantOp.InputCreator.AppendOverrides(
			BusolaMigratorComponentName,
			[]*gqlschema.ConfigEntryInput{
				{
					Key:   "deployment.env.instanceID",
					Value: wantOp.InstanceID,
				},
				{
					Key:   "global.istio.gateway.name",
					Value: "kyma-gateway",
				},
			})

		step := NewBusolaMigratorOverridesStep()
		gotOp, repeat, err := step.Run(wantOp, logrus.New())

		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.EqualValues(t, wantOp, gotOp)
	})
}
