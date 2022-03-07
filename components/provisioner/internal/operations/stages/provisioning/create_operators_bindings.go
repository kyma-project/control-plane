package provisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

const (
	l2OperatorClusterRoleBindingName            = "l2-operator-view"
	l3OperatorClusterRoleBindingName            = "l3-operator-admin"
	administratorOperatorClusterRoleBindingName = "administrator"

	l2OperatorClusterRoleBindingRoleRefName = "view"
	l3OperatorClusterRoleBindingRoleRefName = "cluster-admin"
	ownerClusterRoleBindingRoleRefName      = "cluster-admin"

	groupKindSubject = "Group"
	userKindSubject  = "User"
)

type OperatorRoleBinding struct {
	L2SubjectName    string `envconfig:"default=runtimeOperator"`
	L3SubjectName    string `envconfig:"default=runtimeAdmin"`
	CreatingForAdmin bool   `envconfig:"default=false"`
}

type CreateBindingsForOperatorsStep struct {
	k8sClientProvider         k8s.K8sClientProvider
	operatorRoleBindingConfig OperatorRoleBinding
	nextStep                  model.OperationStage
	timeLimit                 time.Duration
}

func NewCreateBindingsForOperatorsStep(
	k8sClientProvider k8s.K8sClientProvider,
	operatorRoleBindingConfig OperatorRoleBinding,
	nextStep model.OperationStage,
	timeLimit time.Duration) *CreateBindingsForOperatorsStep {

	return &CreateBindingsForOperatorsStep{
		k8sClientProvider:         k8sClientProvider,
		operatorRoleBindingConfig: operatorRoleBindingConfig,
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
	if cluster.Kubeconfig == nil {
		return operations.StageResult{}, operations.ErrKubeconfigNil
	}

	k8sClient, err := s.k8sClientProvider.CreateK8SClient(*cluster.Kubeconfig)
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("failed to create k8s client: %v", err)
	}

	clusterRoleBindings := make([]v12.ClusterRoleBinding, 0)

	clusterRoleBindings = append(clusterRoleBindings,
		buildClusterRoleBinding(
			l2OperatorClusterRoleBindingName,
			s.operatorRoleBindingConfig.L2SubjectName,
			l2OperatorClusterRoleBindingRoleRefName,
			groupKindSubject,
			map[string]string{"app": "kyma"}))
	clusterRoleBindings = append(clusterRoleBindings,
		buildClusterRoleBinding(
			l3OperatorClusterRoleBindingName,
			s.operatorRoleBindingConfig.L3SubjectName,
			l3OperatorClusterRoleBindingRoleRefName,
			groupKindSubject,
			map[string]string{"app": "kyma"}))

	if s.operatorRoleBindingConfig.CreatingForAdmin {
		for i, administrator := range cluster.Administrators {
			clusterRoleBindings = append(clusterRoleBindings,
				buildClusterRoleBinding(
					fmt.Sprintf("%s%d", administratorOperatorClusterRoleBindingName, i),
					administrator,
					ownerClusterRoleBindingRoleRefName,
					userKindSubject,
					map[string]string{"app": "kyma", "type": "admin"}))
		}
	}
	if err := k8sClient.RbacV1().ClusterRoleBindings().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "type=admin"}); err != nil {
		return operations.StageResult{}, fmt.Errorf("failed to delete cluster role bindings: %v", err)
	}

	if err := createClusterRoleBindings(k8sClient.RbacV1().ClusterRoleBindings(), clusterRoleBindings...); err != nil {
		return operations.StageResult{}, fmt.Errorf("failed to create cluster role bindings: %v", err)
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
			if !errors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create %s ClusterRoleBinding: %v", crb.Name, err)
			}
		}
	}
	return nil
}
