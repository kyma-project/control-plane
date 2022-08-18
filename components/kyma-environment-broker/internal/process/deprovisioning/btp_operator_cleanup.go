package deprovisioning

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	k8serrors2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorBinding         = "ServiceBinding"
)

type BTPOperatorCleanupStep struct {
	operationManager  *process.DeprovisionOperationManager
	provisionerClient provisioner.Client
	k8sClientProvider func(kcfg string) (client.Client, error)
}

func NewBTPOperatorCleanupStep(os storage.Operations, provisionerClient provisioner.Client, k8sClientProvider func(kcfg string) (client.Client, error)) *BTPOperatorCleanupStep {
	return &BTPOperatorCleanupStep{
		operationManager:  process.NewDeprovisionOperationManager(os),
		provisionerClient: provisionerClient,
		k8sClientProvider: k8sClientProvider,
	}
}

func (s *BTPOperatorCleanupStep) Name() string {
	return "BTPOperator_Cleanup"
}

func (s *BTPOperatorCleanupStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if !operation.Temporary {
		log.Info("cleanup executed only for suspensions")
		return operation, 0, nil
	}
	if operation.RuntimeID == "" {
		log.Info("instance has been deprovisioned already")
		return operation, 0, nil
	}
	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		if s.isNotFoundErr(err) {
			log.Info("instance not found in provisioner")
			return operation, 0, nil
		}
		return handleError(s.Name(), operation, err, log, "call to provisioner RuntimeStatus failed")
	}
	if status.RuntimeConfiguration.Kubeconfig == nil {
		err := kebError.NewTemporaryError("empty kubeconfig")
		return handleError(s.Name(), operation, err, log, "provisioner returned empty kubeconfig")
	}
	k := *status.RuntimeConfiguration.Kubeconfig
	hash := sha256.Sum256([]byte(k))
	log.Infof("kubeconfig details length: %v, sha256: %v", len(k), string(hash[:]))
	if len(k) < 10 {
		err := kebError.NewTemporaryError("kubeconfig suspiciously small, requeuing")
		return handleError(s.Name(), operation, err, log, "provisioner returned wrong kubeconfig")
	}
	cli, err := s.k8sClientProvider(k)
	if err != nil {
		err = kebError.AsTemporaryError(err, "failed to create k8s client from the kubeconfig")
		return handleError(s.Name(), operation, err, log, "could not create a k8s client")
	}
	if err := s.deleteServiceBindingsAndInstances(cli, log); err != nil {
		err = kebError.AsTemporaryError(err, "failed BTP operator resource cleanup")
		return handleError(s.Name(), operation, err, log, "could not delete bindings and service instances")
	}
	return operation, 0, nil
}

func (s *BTPOperatorCleanupStep) deleteServiceBindingsAndInstances(k8sClient client.Client, log logrus.FieldLogger) error {
	namespaces := corev1.NamespaceList{}
	if err := k8sClient.List(context.Background(), &namespaces); err != nil {
		return err
	}
	requeue := s.deleteResource(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorBinding}, log)
	requeue = requeue || s.deleteResource(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorServiceInstance}, log)
	if requeue {
		return fmt.Errorf("waiting for resources to be deleted")
	}
	return nil
}

func (s *BTPOperatorCleanupStep) deleteResource(k8sClient client.Client, namespaces corev1.NamespaceList, gvk schema.GroupVersionKind, log logrus.FieldLogger) (requeue bool) {
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	stillExistingCount := 0
	for _, ns := range namespaces.Items {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(listGvk)
		if err := k8sClient.List(context.Background(), list, client.InNamespace(ns.Name)); err != nil {
			log.Errorf("failed listing resource %v in namespace %v", gvk, ns.Name)
			if k8serrors2.IsNoMatchError(err) {
				// CRD doesn't exist anymore
				return false
			}
			requeue = true
		}
		stillExistingCount += len(list.Items)
	}
	if stillExistingCount == 0 {
		return
	}
	requeue = true
	for _, ns := range namespaces.Items {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := k8sClient.DeleteAllOf(context.Background(), obj, client.InNamespace(ns.Name)); err != nil {
			log.Errorf("failed deleting resources %v in namespace %v", gvk, ns.Name)
		}
	}
	return
}

func (s *BTPOperatorCleanupStep) isNotFoundErr(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
