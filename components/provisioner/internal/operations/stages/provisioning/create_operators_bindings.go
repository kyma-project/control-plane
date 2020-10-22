package provisioning

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"time"
)

type CreateBindingsForOperatorsStep struct {
	k8sClientProvider k8s.K8sClientProvider
	nextStep          model.OperationStage
	timeLimit         time.Duration
}

func NewCreateBindingsForOperatorsStep(k8sClientProvider k8s.K8sClientProvider, nextStep model.OperationStage, timeLimit time.Duration) *CreateBindingsForOperatorsStep {
	return &CreateBindingsForOperatorsStep{
		k8sClientProvider: k8sClientProvider,
		nextStep:          nextStep,
		timeLimit:         timeLimit,
	}
}

func (s *CreateBindingsForOperatorsStep) Name() model.OperationStage {
	return model.WaitingForInstallation
}

func (s *CreateBindingsForOperatorsStep) TimeLimit() time.Duration {
	return s.timeLimit
}

// TODO: Add tests!
func (s *CreateBindingsForOperatorsStep) Run(cluster model.Cluster, _ model.Operation, _ logrus.FieldLogger) (operations.StageResult, error) {
	if cluster.Kubeconfig == nil {
		return operations.StageResult{}, fmt.Errorf("cluster kubeconfig is nil")
	}

	k8sClient, err := s.k8sClientProvider.CreateK8SClient(*cluster.Kubeconfig)
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("failed to create k8s client: %v", err)
	}

	// TODO: Parametrize these. At least the subjectNames
	l2OperatorView := buildClusterRoleBinding("l2-operator-view", "runtimeOperator", "view")
	l3OperatorAdmin := buildClusterRoleBinding("l3-operator-admin", "runtimeAdmin", "cluster-admin")
	if err := createClusterRoleBindings(k8sClient.RbacV1().ClusterRoleBindings(), l2OperatorView, l3OperatorAdmin); err != nil {
		return operations.StageResult{}, fmt.Errorf("failed to create cluster role bindings: %v", err)
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}

func buildClusterRoleBinding(metaName, subjectName, roleRefName string) v12.ClusterRoleBinding {
	return v12.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   metaName,
			Labels: map[string]string{"app": "kyma"},
		},
		Subjects: []v12.Subject{{
			Kind:     "Group",
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
			// TODO: Check if error is "already exists". If so it should be omitted
			return fmt.Errorf("failed to create %s ClusterRoleBinding: %v", crb.Name, err)
		}
	}
	return nil
}
