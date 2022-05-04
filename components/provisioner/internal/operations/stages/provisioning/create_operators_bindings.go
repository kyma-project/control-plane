package provisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

const (
	l2OperatorClusterRoleBindingName            = "l2-operator"
	l3OperatorClusterRoleBindingName            = "l3-operator-admin"
	administratorOperatorClusterRoleBindingName = "administrator"

	l3OperatorClusterRoleBindingRoleRefName = "cluster-admin"
	ownerClusterRoleBindingRoleRefName      = "cluster-admin"

	l2OperatorClusterRoleName       = "l2-operator"
	l2OperatorRulesClusterRoleName  = "l2-operator-rules"
	l2OperatorBaseRolesLabelKey     = "rbac.authorization.k8s.io/aggregate-to-edit"
	l2OperatorExtendedRolesLabelKey = "rbac.authorization.k8s.io/aggregate-to-l2-operator"

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
		return operations.StageResult{}, err.Append("failed to create k8s client").SetComponent(apperrors.ErrClusterK8SClient)
	}

	clusterRoles := make([]v12.ClusterRole, 0)
	clusterRoles = append(clusterRoles,
		buildClusterRole(
			l2OperatorClusterRoleName,
			map[string]string{"app": "kyma"},
			[]metav1.LabelSelector{
				{MatchLabels: map[string]string{l2OperatorBaseRolesLabelKey: "true"}},
				{MatchLabels: map[string]string{l2OperatorExtendedRolesLabelKey: "true"}},
			},
			nil,
		),
	)
	clusterRoles = append(clusterRoles,
		buildClusterRole(
			l2OperatorRulesClusterRoleName,
			map[string]string{"app": "kyma", l2OperatorExtendedRolesLabelKey: "true"},
			nil,
			[]v12.PolicyRule{
				{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"get", "list", "watch"}},
			},
		),
	)

	clusterRoleBindings := make([]v12.ClusterRoleBinding, 0)

	clusterRoleBindings = append(clusterRoleBindings,
		buildClusterRoleBinding(
			l2OperatorClusterRoleBindingName,
			s.operatorRoleBindingConfig.L2SubjectName,
			l2OperatorClusterRoleBindingName,
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

	if err := createClusterRoles(k8sClient.RbacV1().ClusterRoles(), clusterRoles); err != nil {
		return operations.StageResult{}, err
	}

	if err := k8sClient.RbacV1().ClusterRoleBindings().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "type=admin"}); err != nil {
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

func buildClusterRole(name string, labels map[string]string, aggregationSelectors []metav1.LabelSelector, rules []v12.PolicyRule) v12.ClusterRole {

	cr := v12.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	if len(aggregationSelectors) > 0 {
		cr.AggregationRule = &v12.AggregationRule{
			ClusterRoleSelectors: aggregationSelectors,
		}
	}

	if len(rules) > 0 {
		cr.Rules = rules
	}

	return cr
}

func createClusterRoles(crClient v1.ClusterRoleInterface, clusterRoles []v12.ClusterRole) error {
	for _, cr := range clusterRoles {
		if _, err := crClient.Create(context.Background(), &cr, metav1.CreateOptions{}); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return util.K8SErrorToAppError(errors.Wrapf(err, "failed to create %s ClusterRole", cr.Name)).SetComponent(apperrors.ErrClusterK8SClient)
			}
		}
	}

	return nil
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
