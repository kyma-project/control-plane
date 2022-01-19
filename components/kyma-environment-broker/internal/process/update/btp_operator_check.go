package update

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

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

	processMustBeBlocked, err := s.CRDsInstalledByUser(k8sCli)
	if err != nil {
		log.Warnf("Unable to check, if BTP operator CRDs exists: %s", err.Error())
		return operation, time.Minute, nil
	}
	if processMustBeBlocked {
		return s.operationManager.OperationFailed(operation, "BTP Operator already exists", log)
	}

	return operation, 0, nil
}

func (s *BTPOperatorCheckStep) CRDsInstalledByUser(c client.Client) (bool, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	err := c.Get(context.Background(), client.ObjectKey{Name: "servicebindings.services.cloud.sap.com"}, crd)
	if err == nil {
		if !s.managedByReconciler(crd) {
			return true, nil
		}
	}
	if !errors.IsNotFound(err) {
		return false, err
	}

	err = c.Get(context.Background(), client.ObjectKey{Name: "serviceinstances.services.cloud.sap.com"}, crd)
	if err == nil {
		if !s.managedByReconciler(crd) {
			return true, nil
		}
	}
	if !errors.IsNotFound(err) {
		return false, err
	}

	return false, nil
}

func (s *BTPOperatorCheckStep) managedByReconciler(crd *apiextensions.CustomResourceDefinition) bool {
	_, found := crd.Labels["reconciler.kyma-project.io/managed-by"]
	return found
}
