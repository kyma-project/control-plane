package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
)

type RuntimeTagsStep struct {
	internalEvalUpdater *InternalEvalUpdater
	provisionerClient   provisioner.Client
}

// ensure the interface is implemented
var _ process.Step = (*RuntimeTagsStep)(nil)

func NewRuntimeTagsStep(internalEvalUpdater *InternalEvalUpdater, provisionerClient provisioner.Client) *RuntimeTagsStep {
	return &RuntimeTagsStep{
		internalEvalUpdater: internalEvalUpdater,
		provisionerClient:   provisionerClient,
	}
}
func (e *RuntimeTagsStep) Name() string {
	return "AVS_Tags"
}

func (s *RuntimeTagsStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		return operation, 1 * time.Minute, err
	}

	tags := []*avs.Tag{
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Name),
			TagClassId: s.internalEvalUpdater.avsConfig.GardenerShootNameTagClassId,
		},
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Seed),
			TagClassId: s.internalEvalUpdater.avsConfig.GardenerSeedNameTagClassId,
		},
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Region),
			TagClassId: s.internalEvalUpdater.avsConfig.RegionTagClassId,
		},
	}

	operation, repeat, err := s.internalEvalUpdater.AddTagsToEval(tags, operation, "", log)
	if err != nil || repeat != 0 {
		log.Errorf("while adding Tags to Evaluation: %s", err)
		return operation, repeat, nil
	}

	return operation, 0 * time.Second, nil
}
