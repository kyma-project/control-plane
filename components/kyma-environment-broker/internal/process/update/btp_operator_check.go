package update

import (
	"context"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BTPOperatorCheckStep struct {
	operationManager *process.UpdateOperationManager
}

func NewBTPOperatorCheckStep(os storage.Operations) *BTPOperatorCheckStep {
	return &BTPOperatorCheckStep{
		operationManager: process.NewUpdateOperationManager(os),
	}
}

func (s *BTPOperatorCheckStep) Name() string {
	return "BTPOperatorCheck"
}

func (s *BTPOperatorCheckStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	if operation.InstanceDetails.SCMigrationTriggered {
		return operation, 0, nil
	}
	if operation.Kubeconfig == "" {
		log.Errorf("Kubeconfig must not be empty")
		return s.operationManager.OperationFailed(operation, "Kubeconfig is not present", log)
	}
	restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(operation.Kubeconfig))
	if err != nil {
		log.Errorf("Unable to create rest config: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "Unable to create rest config", log)
	}

	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: scheme.Scheme,
	})

	btpOperatorCRDsExists, err := s.CRDsExists(k8sCli)
	if err != nil {
		log.Warnf("Unable to check, if BTP operator CRDs exists: %s", err.Error())
		return operation, time.Minute, nil
	}
	if btpOperatorCRDsExists {
		return s.operationManager.OperationFailed(operation, "BTP Operartor already exists", log)
	}

	return operation, 0, nil
}

func (s *BTPOperatorCheckStep) CRDsExists(c client.Client) (bool, error) {
	crdsList := &apiextensions.CustomResourceDefinitionList{}

	err := c.List(context.Background(), crdsList)
	if err != nil {
		return false, err
	}

	for _, crd := range crdsList.Items {
		if strings.Contains(crd.Spec.Group, "services.cloud.sap.com") {
			if crd.Spec.Names.Kind == "ServiceBinding" || crd.Spec.Names.Kind == "ServiceInstance" {
				return true, nil
			}
		}
	}
	return false, nil
}
