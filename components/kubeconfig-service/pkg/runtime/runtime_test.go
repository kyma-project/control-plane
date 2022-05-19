package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var sa1 = corev1.ServiceAccount{
	ObjectMeta: v1.ObjectMeta{
		Name:      "sa1",
		Namespace: "default",
	},
}

func NewRuntimeClientTest(kubeConfig []byte, userID string, L2L3OperatiorRole string) (*RuntimeClient, error) {
	clientset := fake.NewSimpleClientset()

	user := SAInfo{
		ServiceAccountName:     userID,
		ClusterRoleName:        userID,
		ClusterRoleBindingName: userID,
		Namespace:              "default",
	}
	rollbackE := RollbackE{}
	return &RuntimeClient{clientset, user, L2L3OperatiorRole, rollbackE}, nil
}

func TestCreateserviceaccount(t *testing.T) {
	var expectedSaName, expectedNamespace = "sa1", "default"
	t.Run("If no service account exists one is created", func(t *testing.T) {
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeAdmin")
		assert.NoError(t, err)
		err = rtc.createServiceAccount()
		assert.Nil(t, err)

		sa, err := rtc.K8s.CoreV1().ServiceAccounts(rtc.User.Namespace).Get(context.TODO(), rtc.User.ServiceAccountName, v1.GetOptions{})
		assert.NotNil(t, sa)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, sa.Name)
		assert.Equal(t, expectedNamespace, sa.Namespace)
	})

	t.Run("If service account already exists nothing is created", func(t *testing.T) {
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeAdmin")
		assert.NoError(t, err)

		_, err = rtc.K8s.CoreV1().ServiceAccounts(sa1.Namespace).Create(context.TODO(), &sa1, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error occurred when creating ServiceAccount: %v", err)
		}

		err = rtc.createServiceAccount()
		assert.Nil(t, err)

		sa, err := rtc.K8s.CoreV1().ServiceAccounts(rtc.User.Namespace).Get(context.TODO(), rtc.User.ServiceAccountName, v1.GetOptions{})
		assert.NotNil(t, sa)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, sa.Name)
		assert.Equal(t, expectedNamespace, sa.Namespace)
	})

	t.Run("If no clusterrole exists one is created", func(t *testing.T) {
		var clusterRoleName = "sa1"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator")
		assert.NoError(t, err)

		err = rtc.createClusterRole()
		assert.Nil(t, err)

		crClient := rtc.K8s.RbacV1().ClusterRoles()
		cr, err := crClient.Get(context.TODO(), rtc.User.ClusterRoleName, v1.GetOptions{})
		assert.NotNil(t, cr)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleName, cr.Name)
	})

	t.Run("If input clusterrole not supported no one is created", func(t *testing.T) {
		//`unSupportedOperation` not belong to `runtimeAdmin`/`runtimeOperator`
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "unSupportedOperation")
		assert.NoError(t, err)

		err = rtc.createClusterRole()
		assert.Error(t, err)

		crClient := rtc.K8s.RbacV1().ClusterRoles()
		cr, err := crClient.Get(context.TODO(), rtc.User.ClusterRoleName, v1.GetOptions{})
		assert.Nil(t, cr)
		assert.Error(t, err)
	})

	t.Run("If no clusterrole exists one is created", func(t *testing.T) {
		var clusterRoleBindingName = "sa1"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator")
		assert.NoError(t, err)

		err = rtc.createClusterRoleBinding()
		assert.Nil(t, err)

		crbClient := rtc.K8s.RbacV1().ClusterRoleBindings()
		crb, err := crbClient.Get(context.TODO(), rtc.User.ClusterRoleBindingName, v1.GetOptions{})
		assert.NotNil(t, crb)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleBindingName, crb.Name)
	})
}
