package provisioning

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	provisioning_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const dynamicKubeconfig = "dynamic_kubeconfig"

func TestCreateBindingsForOperatorsStep_Run(t *testing.T) {

	cluster := model.Cluster{
		Kubeconfig: util.StringPtr("kubeconfig"),
		ClusterConfig: model.GardenerConfig{
			Name: "shoot",
		},
	}

	operatorBindingConfig := OperatorRoleBinding{
		L2SubjectName: "l2name",
		L3SubjectName: "l3name",
	}

	dynamicKubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}
	dynamicKubeconfigProvider.On("FetchFromGardener", "shoot").Return([]byte("dynamic_kubeconfig"), nil)

	t.Run("should return next step when finished", func(t *testing.T) {
		// given
		k8sClient := fake.NewSimpleClientset()
		k8sClientProvider := &mocks.K8sClientProvider{}
		k8sClientProvider.On("CreateK8SClient", dynamicKubeconfig).Return(k8sClient, nil)

		step := NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		result, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.NoError(t, err)
		assert.Equal(t, nextStageName, result.Stage)
		assert.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should not fail if cluster role already exists", func(t *testing.T) {
		// given
		k8sClient := fake.NewSimpleClientset()
		clusterRole := buildClusterRole(l2OperatorClusterRoleName, map[string]string{"app": "kyma"}, []metav1.LabelSelector{
			{MatchLabels: map[string]string{l2OperatorBaseRolesLabelKey: "true"}},
			{MatchLabels: map[string]string{l2OperatorExtendedRolesLabelKey: "true"}},
		}, nil)
		_, err := k8sClient.RbacV1().ClusterRoles().Create(context.Background(), &clusterRole, metav1.CreateOptions{})
		require.NoError(t, err)

		k8sClientProvider := &mocks.K8sClientProvider{}
		k8sClientProvider.On("CreateK8SClient", dynamicKubeconfig).Return(k8sClient, nil)

		step := NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		result, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.NoError(t, err)
		assert.Equal(t, nextStageName, result.Stage)
		assert.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should not fail if cluster role binding already exists", func(t *testing.T) {
		// given
		k8sClient := fake.NewSimpleClientset()
		clusterRoleBinding := buildClusterRoleBinding(l2OperatorClusterRoleBindingName, operatorBindingConfig.L2SubjectName, l2OperatorClusterRoleName, groupKindSubject, map[string]string{"app": "kyma"})
		_, err := k8sClient.RbacV1().ClusterRoleBindings().Create(context.Background(), &clusterRoleBinding, metav1.CreateOptions{})
		require.NoError(t, err)

		k8sClientProvider := &mocks.K8sClientProvider{}
		k8sClientProvider.On("CreateK8SClient", dynamicKubeconfig).Return(k8sClient, nil)

		step := NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		result, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.NoError(t, err)
		assert.Equal(t, nextStageName, result.Stage)
		assert.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should return error when failed to get dynamic kubeconfig", func(t *testing.T) {
		// given
		dynamicKubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}
		dynamicKubeconfigProvider.On("FetchFromGardener", "shoot").Return(nil, errors.New("some error"))

		step := NewCreateBindingsForOperatorsStep(nil, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		_, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.Error(t, err)
	})

	t.Run("should return error when failed to provide k8s client", func(t *testing.T) {
		// given
		k8sClientProvider := &mocks.K8sClientProvider{}
		k8sClientProvider.On("CreateK8SClient", dynamicKubeconfig).Return(nil, apperrors.Internal("error"))

		step := NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		_, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.Error(t, err)
	})

	t.Run("should return error when failed to create cluster role binding", func(t *testing.T) {
		// given
		k8sClient := fake.NewSimpleClientset()
		k8sClient.Fake.PrependReactor(
			"*",
			"*",
			func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, fmt.Errorf("error")
			})

		k8sClientProvider := &mocks.K8sClientProvider{}
		k8sClientProvider.On("CreateK8SClient", dynamicKubeconfig).Return(k8sClient, nil)

		step := NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorBindingConfig, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		_, err := step.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.Error(t, err)
	})
}
