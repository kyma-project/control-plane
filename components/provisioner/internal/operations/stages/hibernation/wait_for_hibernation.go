package hibernation

import (
	"context"
	"time"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate mockery -name=GardenerClient
type GardenerClient interface {
	Get(ctx context.Context, name string, options v1.GetOptions) (*gardener_types.Shoot, error)
}

type WaitForHibernation struct {
	gardenerClient GardenerClient
	nextStep       model.OperationStage
	timeLimit      time.Duration
}

func NewWaitForHibernation(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForHibernation {
	return &WaitForHibernation{
		gardenerClient: gardenerClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (c *WaitForHibernation) Name() model.OperationStage {
	return model.WaitForHibernation
}

func (c *WaitForHibernation) TimeLimit() time.Duration {
	return c.timeLimit
}

func (c *WaitForHibernation) Run(cluster model.Cluster, operation model.Operation, _ logrus.FieldLogger) (operations.StageResult, error) {

	shoot, err := c.gardenerClient.Get(context.Background(), cluster.ClusterConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, err
	}

	if shoot.Status.IsHibernated {
		return operations.StageResult{
			Stage: c.nextStep,
			Delay: 0,
		}, nil
	}

	return operations.StageResult{
		Stage: c.Name(),
		Delay: 30 * time.Second,
	}, nil
}
