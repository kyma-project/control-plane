package provisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"

	core "k8s.io/api/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	administratorOperatorClusterRoleBindingName = "administrator"

	ownerClusterRoleBindingRoleRefName = "cluster-admin"

	userKindSubject = "User"
)

//go:generate mockery --name=DynamicKubeconfigProvider
type DynamicKubeconfigProvider interface {
	FetchFromRequest(shootName string) ([]byte, error)
}

type OperatorRoleBinding struct {
	CreatingForAdmin bool `envconfig:"default=false"`
}

type CreateBindingsForOperatorsStep struct {
	k8sClientProvider         k8s.K8sClientProvider
	operatorRoleBindingConfig OperatorRoleBinding
	dynamicKubeconfigProvider DynamicKubeconfigProvider
	nextStep                  model.OperationStage
	timeLimit                 time.Duration
}

func NewCreateBindingsForOperatorsStep(
	k8sClientProvider k8s.K8sClientProvider,
	operatorRoleBindingConfig OperatorRoleBinding,
	dynamicKubeconfigProvider DynamicKubeconfigProvider,
	nextStep model.OperationStage,
	timeLimit time.Duration) *CreateBindingsForOperatorsStep {

	return &CreateBindingsForOperatorsStep{
		k8sClientProvider:         k8sClientProvider,
		operatorRoleBindingConfig: operatorRoleBindingConfig,
		dynamicKubeconfigProvider: dynamicKubeconfigProvider,
		nextStep:                  nextStep,
		timeLimit:                 timeLimit,
	}
}

func (s *CreateBindingsForOperatorsStep) Name() model.OperationStage {
	return model.CreatingBindingsForOperators
}

func (s *CreateBindingsForOperatorsStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *CreateBindingsForOperatorsStep) Run(cluster model.Cluster, _ model.Operation, log logrus.FieldLogger) (operations.StageResult, error) {

	var kubeconfig []byte
	{
		var err error
		kubeconfig, err = s.dynamicKubeconfigProvider.FetchFromRequest(cluster.ClusterConfig.Name)
		if err != nil {
			// we cannot read kubeconfig from gardener cluster
			return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
		}
	}

	k8sClient, err := s.k8sClientProvider.CreateK8SClient(string(kubeconfig))
	if err != nil {
		return operations.StageResult{}, err.Append("failed to create k8s client").SetComponent(apperrors.ErrClusterK8SClient)
	}

	if err := s.createNamespace(k8sClient.CoreV1().Namespaces(), "istio-system"); err != nil {
		return operations.StageResult{}, err
	}

	clusterRoleBindings := make([]v12.ClusterRoleBinding, 0)

	if s.operatorRoleBindingConfig.CreatingForAdmin {
		for i, administrator := range cluster.Administrators {
			clusterRoleBindings = append(clusterRoleBindings,
				buildClusterRoleBinding(
					fmt.Sprintf("%s%d", administratorOperatorClusterRoleBindingName, i),
					administrator,
					ownerClusterRoleBindingRoleRefName,
					userKindSubject,
					map[string]string{
						"app":                                   "kyma",
						"reconciler.kyma-project.io/managed-by": "reconciler",
					}))
		}
	}

	if err := k8sClient.RbacV1().ClusterRoleBindings().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "reconciler.kyma-project.io/managed-by=reconciler,app=kyma"}); err != nil {
		return operations.StageResult{}, util.K8SErrorToAppError(errors.Wrap(err, "failed to delete cluster role bindings")).SetComponent(apperrors.ErrClusterK8SClient)
	}

	if err := createClusterRoleBindings(k8sClient.RbacV1().ClusterRoleBindings(), clusterRoleBindings...); err != nil {
		return operations.StageResult{}, err
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}

func buildClusterRoleBinding(metaName, subjectName, roleRefName, subjectKind string, labels map[string]string) v12.ClusterRoleBinding {
	return v12.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   metaName,
			Labels: labels,
		},
		Subjects: []v12.Subject{{
			Kind:     subjectKind,
			Name:     subjectName,
			APIGroup: "rbac.authorization.k8s.io",
		}},
		RoleRef: v12.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleRefName,
		},
	}
}

func createClusterRoleBindings(crbClient v1.ClusterRoleBindingInterface, clusterRoleBindings ...v12.ClusterRoleBinding) error {
	for _, crb := range clusterRoleBindings {
		if _, err := crbClient.Create(context.Background(), &crb, metav1.CreateOptions{}); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return util.K8SErrorToAppError(errors.Wrapf(err, "failed to create %s ClusterRoleBinding", crb.Name)).SetComponent(apperrors.ErrClusterK8SClient)
			}
		}
	}
	return nil
}

func (c *CreateBindingsForOperatorsStep) createNamespace(namespaceInterface v1core.NamespaceInterface, namespace string) apperrors.AppError {
	ns := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	_, err := namespaceInterface.Create(context.Background(), ns, metav1.CreateOptions{})

	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return util.K8SErrorToAppError(errors.Wrap(err, "Failed to create namespace"))
	}
	return nil
}
