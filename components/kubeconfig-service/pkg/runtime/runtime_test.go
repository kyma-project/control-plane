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

func TestCreateserviceaccount(t *testing.T) {
	user := SAInfo{
		ServiceAccountName:     "sa1",
		ClusterRoleName:        "sa1",
		ClusterRoleBindingName: "sa1",
		Namespace:              "default",
	}
	var expectedSaName, expectedNamespace = "sa1", "default"
	t.Run("If no service account exists one is created", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		err := createServiceAccount(clientset, user)
		assert.Nil(t, err)

		sa, err := clientset.CoreV1().ServiceAccounts(user.Namespace).Get(context.TODO(), user.ServiceAccountName, v1.GetOptions{})
		assert.NotNil(t, sa)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, sa.Name)
		assert.Equal(t, expectedNamespace, sa.Namespace)
	})

	t.Run("If service account already exists nothing is created", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		_, err := clientset.CoreV1().ServiceAccounts(sa1.Namespace).Create(context.TODO(), &sa1, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error occurred when creating ServiceAccount: %v", err)
		}

		err = createServiceAccount(clientset, user)
		assert.Nil(t, err)

		sa, err := clientset.CoreV1().ServiceAccounts(user.Namespace).Get(context.TODO(), user.ServiceAccountName, v1.GetOptions{})
		assert.NotNil(t, sa)
		assert.NoError(t, err)
		assert.Equal(t, expectedSaName, sa.Name)
		assert.Equal(t, expectedNamespace, sa.Namespace)
	})

	t.Run("If no clusterrole exists one is created", func(t *testing.T) {
		var clusterRoleName = "user123"
		clientset := fake.NewSimpleClientset()
		err := createClusterRole(clientset, clusterRoleName, "runtimeAdmin")
		assert.Nil(t, err)

		crClient := clientset.RbacV1().ClusterRoles()
		cr, err := crClient.Get(context.TODO(), clusterRoleName, v1.GetOptions{})
		assert.NotNil(t, cr)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleName, cr.Name)
	})

	t.Run("If no clusterrole exists one is created", func(t *testing.T) {
		var clusterRoleBindingName = "sa1"
		clientset := fake.NewSimpleClientset()
		err := createClusterRoleBinding(clientset, user)
		//assert.True(t, created)
		assert.Nil(t, err)

		crbClient := clientset.RbacV1().ClusterRoleBindings()
		crb, err := crbClient.Get(context.TODO(), user.ClusterRoleBindingName, v1.GetOptions{})
		assert.NotNil(t, crb)
		assert.NoError(t, err)
		assert.Equal(t, clusterRoleBindingName, crb.Name)
	})
}
