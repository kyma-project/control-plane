package update

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/sirupsen/logrus"
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
	if operation.K8sClient == nil {
		log.Errorf("k8s client must be provided")
		return s.operationManager.OperationFailed(operation, "k8s client must be provided", log)
	}
	processMustBeBlocked, err := s.CRDsInstalledByUser(operation.K8sClient)
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
