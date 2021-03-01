package test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/test/steps"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Test_ProvisioningProcess(t *testing.T) {
	t.Run("Operaction succeeded", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		logs := logrus.New()

		provisionManager := provisioning.NewManager(db.Operations(), logs)
		provisioningSteps := []struct {
			disabled bool
			weight   int
			step     provisioning.Step
		}{
			{
				weight: 1,
				step:   &steps.StepOne{},
			},
			{
				weight: 2,
				step:   &steps.StepTwo{},
			},
			{
				weight: 2,
				step:   &steps.StepThree{},
			},
			{
				weight: 3,
				step:   &steps.StepFour{},
			},
			{
				weight: 3,
				step:   &steps.StepFive{},
			},
		}
		for _, step := range provisioningSteps {
			if !step.disabled {
				provisionManager.AddStep(step.weight, step.step)
			}
		}

		provisionQueue := process.NewQueue(provisionManager, logs)
		provisionQueue.Run(context.TODO().Done(), 5)

		logs.Info("Add operation to queue")
		operation, err := internal.NewProvisioningOperationWithID(
			"7b2de34d-de09-4993-afde-d4f102cbcbc4",
			"c434aa97-a0da-4fb5-844c-7b1da72d429e",
			internal.ProvisioningParameters{})
		err = db.Operations().InsertProvisioningOperation(operation)
		require.NoError(t, err)

		provisionQueue.Add(operation.ID)

		err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (done bool, err error) {
			op, err := db.Operations().GetOperationByID("7b2de34d-de09-4993-afde-d4f102cbcbc4")
			if err != nil {
				logs.Errorf("cannot get operation")
				return false, nil
			}
			if op.State == domain.Succeeded {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err)
	})
}
