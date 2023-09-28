package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime"
	"github.com/sirupsen/logrus"
)

type ConnectAgentStep struct {
	runtimeConfigurator       runtime.Configurator
	dynamicKubeconfigProvider DynamicKubeconfigProvider
	nextStage                 model.OperationStage
	timeLimit                 time.Duration
}

func NewConnectAgentStep(
	configurator runtime.Configurator,
	dynamicKubeconfigProvider DynamicKubeconfigProvider,
	nextStage model.OperationStage,
	timeLimit time.Duration) *ConnectAgentStep {
	return &ConnectAgentStep{
		runtimeConfigurator:       configurator,
		dynamicKubeconfigProvider: dynamicKubeconfigProvider,
		nextStage:                 nextStage,
		timeLimit:                 timeLimit,
	}
}

func (s *ConnectAgentStep) Name() model.OperationStage {
	return model.ConnectRuntimeAgent
}

func (s *ConnectAgentStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *ConnectAgentStep) Run(cluster model.Cluster, _ model.Operation, _ logrus.FieldLogger) (operations.StageResult, error) {

	var kubeconfig []byte
	{
		var err error
		kubeconfig, err = s.dynamicKubeconfigProvider.FetchFromRequest(cluster.ClusterConfig.Name)
		if err != nil {
			return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
		}
	}
	err := s.runtimeConfigurator.ConfigureRuntime(cluster, string(kubeconfig))
	if err != nil {
		return operations.StageResult{}, err.Append("failed to configure Runtime Agent")
	}

	return operations.StageResult{Stage: s.nextStage, Delay: 0}, nil
}
