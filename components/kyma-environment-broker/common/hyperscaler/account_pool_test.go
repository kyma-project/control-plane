package hyperscaler

import (
	"context"
	"testing"

	gardener_types "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace = "garden-namespace"
)

func TestCredentialsSecretBinding(t *testing.T) {

	pool := newTestAccountPool()

	var testcases = []struct {
		testDescription           string
		tenantName                string
		hyperscalerType           Type
		expectedSecretBindingName string
		expectedError             string
	}{
		{"In-use credential for tenant1, GCP returns existing secret",
			"tenant1", GCP, "secretBinding1", ""},

		{"In-use credential for tenant1, Azure returns existing secret",
			"tenant1", Azure, "secretBinding2", ""},

		{"In-use credential for tenant2, GCP returns existing secret",
			"tenant2", GCP, "secretBinding3", ""},

		{"Available credential for tenant3, AWS labels and returns existing secret",
			"tenant3", GCP, "secretBinding4", ""},

		{"Available credential for tenant4, GCP labels and returns existing secret",
			"tenant4", AWS, "secretBinding5", ""},

		{"There is only dirty Secret for tenant9, Azure labels and returns a new existing secret",
			"tenant9", Azure, "secretBinding9", ""},

		{"No Available credential for tenant5, Azure returns error",
			"tenant5", Azure, "",
			"failed to find unassigned secret binding for hyperscalerType: azure"},

		{"No Available credential for tenant6, GCP returns error - ignore secret binding with label shared=true",
			"tenant6", GCP, "",
			"failed to find unassigned secret binding for hyperscalerType: gcp"},

		{"Available credential for tenant7, AWS labels and returns existing secret from different namespace",
			"tenant7", AWS, "secretBinding7", ""},

		{"No Available credential for tenant8, AWS returns error - failed to get referenced secret",
			"tenant8", AWS, "",
			"failed to find unassigned secret binding for hyperscalerType: aws"},
	}
	for _, testcase := range testcases {

		t.Run(testcase.testDescription, func(t *testing.T) {
			secretBinding, err := pool.CredentialsSecretBinding(testcase.hyperscalerType, testcase.tenantName)
			actualError := ""
			if err != nil {
				actualError = err.Error()
				assert.Equal(t, testcase.expectedError, actualError)
			} else {
				assert.Equal(t, testcase.expectedSecretBindingName, secretBinding.Name)
				assert.Equal(t, string(testcase.hyperscalerType), secretBinding.Labels["hyperscalerType"])
				assert.Equal(t, testcase.expectedError, actualError)
			}
		})
	}
}

func TestSecretsAccountPool_IsSecretBindingInternal(t *testing.T) {
	t.Run("should return true if internal secret binding found", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithSecretBindingInternal()

		//when
		internal, err := accPool.IsSecretBindingInternal("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.True(t, internal)
	})

	t.Run("should return false if internal secret binding not found", func(t *testing.T) {
		//given
		accPool := newTestAccountPool()

		//when
		internal, err := accPool.IsSecretBindingInternal("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, internal)
	})

	t.Run("should return false when there is no secret binding in the pool", func(t *testing.T) {
		//given
		accPool := newEmptyTestAccountPool()

		//when
		internal, err := accPool.IsSecretBindingInternal("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, internal)
	})
}

func TestSecretsAccountPool_IsSecretBindingDirty(t *testing.T) {
	t.Run("should return true if dirty secret binding found", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithSecretBindingDirty()

		//when
		isdirty, err := accPool.IsSecretBindingDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.True(t, isdirty)
	})

	t.Run("should return false if dirty secret binding not found", func(t *testing.T) {
		//given
		accPool := newTestAccountPool()

		//when
		isdirty, err := accPool.IsSecretBindingDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, isdirty)
	})
}

func TestSecretsAccountPool_IsSecretBindingUsed(t *testing.T) {
	t.Run("should return true when secret binding is in use", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithSingleShoot()

		//when
		used, err := accPool.IsSecretBindingUsed("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.True(t, used)
	})

	t.Run("should return false when secret binding is not in use", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithoutShoots()

		//when
		used, err := accPool.IsSecretBindingUsed("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, used)
	})
}

func TestSecretsAccountPool_MarkSecretBindingAsDirty(t *testing.T) {
	t.Run("should mark secret binding as dirty", func(t *testing.T) {
		//given
		accPool, mockSecretBindings := newTestAccountPoolWithoutShoots()

		//when
		err := accPool.MarkSecretBindingAsDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := mockSecretBindings.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})
}

func newTestAccountPool() AccountPool {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "gcp",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}
	secretBinding2 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding2",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret2",
			Namespace: testNamespace,
		},
	}
	secretBinding3 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding3",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant2",
				"hyperscalerType": "gcp",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret3",
			Namespace: testNamespace,
		},
	}
	secretBinding4 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding4",
			Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "gcp",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret4",
			Namespace: testNamespace,
		},
	}
	secretBinding5 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding5",
			Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "aws",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret5",
			Namespace: testNamespace,
		},
	}
	secretBinding6 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding6",
			Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "gcp",
				"shared":          "true",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret6",
			Namespace: testNamespace,
		},
	}
	secretBinding7 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding7",
			Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "aws",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret7",
			Namespace: "anothernamespace",
		},
	}
	secretBinding8 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding8",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant9",
				"hyperscalerType": "azure",
				"dirty":           "true",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret8",
			Namespace: testNamespace,
		},
	}
	secretBinding9 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding9",
			Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "azure",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret9",
			Namespace: testNamespace,
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1, secretBinding2, secretBinding3, secretBinding4, secretBinding5, secretBinding6, secretBinding7, secretBinding8, secretBinding9).
		CoreV1beta1().SecretBindings(testNamespace)

	return NewAccountPool(gardenerFake, nil)
}

func newTestAccountPoolWithSingleShoot() (AccountPool, v1beta1.SecretBindingInterface) {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secretBinding1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots), mockSecretBindings
}

func newEmptyTestAccountPool() AccountPool {
	secretBinding1 := &gardener_types.SecretBinding{}

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots)
}

func newTestAccountPoolWithSecretBindingInternal() (AccountPool, v1beta1.SecretBindingInterface) {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
				"internal":        "true",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots), mockSecretBindings
}

func newTestAccountPoolWithSecretBindingDirty() (AccountPool, v1beta1.SecretBindingInterface) {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
				"dirty":           "true",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secretBinding1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots), mockSecretBindings
}

func newTestAccountPoolWithShootsUsingSecretBinding() (AccountPool, v1beta1.SecretBindingInterface) {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secretBinding1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	shoot2 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot2",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secretBinding1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, shoot2, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots), mockSecretBindings
}

func newTestAccountPoolWithoutShoots() (AccountPool, v1beta1.SecretBindingInterface) {
	secretBinding1 := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "secretBinding1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      "secret1",
			Namespace: testNamespace,
		},
	}

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockSecretBindings, mockShoots), mockSecretBindings
}
