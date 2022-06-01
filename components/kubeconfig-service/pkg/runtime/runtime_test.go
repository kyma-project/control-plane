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

func NewRuntimeClientTest(kubeConfig []byte, userID string, L2L3OperatiorRole string, tenant string) (*RuntimeClient, error) {
	clientset := fake.NewSimpleClientset()
	coreClientset := fake.NewSimpleClientset()

	user := SAInfo{
		ServiceAccountName:     userID,
		ClusterRoleName:        userID,
		ClusterRoleBindingName: userID,
		Namespace:              "default",
		TenantID:               tenant,
	}
	rollbackE := RollbackE{}
	return &RuntimeClient{clientset, coreClientset, user, L2L3OperatiorRole, rollbackE}, nil
}

func TestCreateserviceaccount(t *testing.T) {
	var expectedSaName, expectedNamespace, expectedTenantID = "sa1", "default", "tenantID"
	t.Run("If no service account exists one is created", func(t *testing.T) {
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeAdmin", "tenantID")
		assert.NoError(t, err)
		err = rtc.createServiceAccount()
		assert.Nil(t, err)

		sa, err := rtc.K8s.CoreV1().ServiceAccounts(rtc.User.Namespace).Get(context.TODO(), rtc.User.ServiceAccountName, v1.GetOptions{})
		assert.NotNil(t, sa)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, sa.Name)
		assert.Equal(t, expectedNamespace, sa.Namespace)
		assert.Equal(t, expectedTenantID, rtc.User.TenantID)
	})

	t.Run("If service account already exists nothing is created", func(t *testing.T) {
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeAdmin", "tenantID")
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
		assert.Equal(t, expectedTenantID, rtc.User.TenantID)
	})

	t.Run("If no clusterrole exists one is created", func(t *testing.T) {
		var clusterRoleName = "sa1"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		err = rtc.createClusterRole()
		assert.Nil(t, err)

		crClient := rtc.K8s.RbacV1().ClusterRoles()
		cr, err := crClient.Get(context.TODO(), rtc.User.ClusterRoleName, v1.GetOptions{})
		assert.NotNil(t, cr)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleName, cr.Name)
		assert.Equal(t, expectedTenantID, rtc.User.TenantID)
	})

	t.Run("If input clusterrole not supported no one is created", func(t *testing.T) {
		//`unSupportedOperation` not belong to `runtimeAdmin`/`runtimeOperator`
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "unSupportedOperation", "tenantID")
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
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		err = rtc.createClusterRoleBinding()
		assert.Nil(t, err)

		crbClient := rtc.K8s.RbacV1().ClusterRoleBindings()
		crb, err := crbClient.Get(context.TODO(), rtc.User.ClusterRoleBindingName, v1.GetOptions{})
		assert.NotNil(t, crb)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleBindingName, crb.Name)
		assert.Equal(t, expectedTenantID, rtc.User.TenantID)
	})

	t.Run("If kcp config map is updated with remove method", func(t *testing.T) {
		var runtimeName = "runtime2"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		configmap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:        "sa1",
				Namespace:   "kcp-system",
				Labels:      map[string]string{"service": "kubeconfig"},
				Annotations: map[string]string{"role": "runtimeOperator", "tenant": "tenantID"},
			},
			Data: map[string]string{"runtime1": "startTime1", "runtime2": "startTime2"},
		}
		cm, err := rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Create(context.Background(), configmap, v1.CreateOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, cm)

		err = rtc.UpdateConfigMap(runtimeName)
		assert.NoError(t, err)

		cm, err = rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Get(context.Background(), "sa1", v1.GetOptions{})
		assert.NotNil(t, cm)
		assert.NoError(t, err)
		assert.Equal(t, "startTime1", cm.Data["runtime1"])
		assert.Empty(t, cm.Data["runtime2"])
	})

	t.Run("If kcp config map deployed when no user exists", func(t *testing.T) {
		var runtimeName, namespaceName = "runtime1", "kcp-system"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		err = rtc.DeployConfigMap(runtimeName, "runtimeOperator")
		assert.NoError(t, err)

		cm, err := rtc.KcpK8s.CoreV1().ConfigMaps(namespaceName).Get(context.Background(), "sa1", v1.GetOptions{})
		assert.NotNil(t, cm)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, cm.Name)
		assert.Equal(t, namespaceName, cm.Namespace)
		assert.Equal(t, "kubeconfig", cm.Labels["service"])
		assert.Equal(t, "runtimeOperator", cm.Annotations["role"])
		assert.Equal(t, expectedTenantID, cm.Annotations["tenant"])
		assert.NotEmpty(t, cm.Data["runtime1"])
	})

	t.Run("If kcp config map deployed when user and runtime existed", func(t *testing.T) {
		var runtimeName, namespaceName = "runtime1", "kcp-system"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		configmap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:        "sa1",
				Namespace:   "kcp-system",
				Labels:      map[string]string{"service": "kubeconfig"},
				Annotations: map[string]string{"role": "runtimeOperator", "tenant": "tenantID"},
			},
			Data: map[string]string{"runtime1": "startTime1"},
		}
		cm, err := rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Create(context.Background(), configmap, v1.CreateOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, cm)

		err = rtc.DeployConfigMap(runtimeName, "runtimeOperator")
		assert.NoError(t, err)

		cm, err = rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Get(context.Background(), "sa1", v1.GetOptions{})
		assert.NotNil(t, cm)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, cm.Name)
		assert.Equal(t, namespaceName, cm.Namespace)
		assert.Equal(t, "kubeconfig", cm.Labels["service"])
		assert.Equal(t, "runtimeOperator", cm.Annotations["role"])
		assert.Equal(t, expectedTenantID, cm.Annotations["tenant"])
		assert.NotEmpty(t, cm.Data["runtime1"])
		assert.NotEqual(t, "startTime1", cm.Data["runtime1"])
	})

	t.Run("If kcp config map deployed when user existed but runtime not", func(t *testing.T) {
		var runtimeName, namespaceName = "runtime2", "kcp-system"
		rtc, err := NewRuntimeClientTest([]byte("kubeconfig"), "sa1", "runtimeOperator", "tenantID")
		assert.NoError(t, err)

		configmap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:        "sa1",
				Namespace:   "kcp-system",
				Labels:      map[string]string{"service": "kubeconfig"},
				Annotations: map[string]string{"role": "runtimeOperator", "tenant": "tenantID"},
			},
			Data: map[string]string{"runtime1": "startTime1"},
		}
		cm, err := rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Create(context.Background(), configmap, v1.CreateOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, cm)

		err = rtc.DeployConfigMap(runtimeName, "runtimeOperator")
		assert.NoError(t, err)

		cm, err = rtc.KcpK8s.CoreV1().ConfigMaps("kcp-system").Get(context.Background(), "sa1", v1.GetOptions{})
		assert.NotNil(t, cm)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, cm.Name)
		assert.Equal(t, namespaceName, cm.Namespace)
		assert.Equal(t, "kubeconfig", cm.Labels["service"])
		assert.Equal(t, "runtimeOperator", cm.Annotations["role"])
		assert.Equal(t, expectedTenantID, cm.Annotations["tenant"])
		assert.Equal(t, "startTime1", cm.Data["runtime1"])
		assert.NotEmpty(t, cm.Data["runtime2"])
	})

}
