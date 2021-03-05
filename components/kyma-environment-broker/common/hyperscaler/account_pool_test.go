package hyperscaler

import (
	"github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "garden-namespace"
)

func TestCredentials(t *testing.T) {

	pool := newTestAccountPool()

	var testcases = []struct {
		testDescription        string
		tenantName             string
		hyperscalerType        Type
		expectedCredentialName string
		expectedError          string
	}{
		{"In-use credential for tenant1, GCP returns existing secret",
			"tenant1", GCP, "secret1", ""},

		{"In-use credential for tenant1, Azure returns existing secret",
			"tenant1", Azure, "secret2", ""},

		{"In-use credential for tenant2, GCP returns existing secret",
			"tenant2", GCP, "secret3", ""},

		{"Available credential for tenant3, AWS labels and returns existing secret",
			"tenant3", GCP, "secret4", ""},

		{"Available credential for tenant4, GCP labels and returns existing secret",
			"tenant4", AWS, "secret5", ""},

		{"No Available credential for tenant5, Azure returns error",
			"tenant5", Azure, "",
			"failed to find unassigned secret binding for hyperscalerType: azure"},

		{"No Available credential for tenant6, GCP returns error - ignore secret binding with label shared=true",
			"tenant6", GCP, "",
			"failed to find unassigned secret binding for hyperscalerType: gcp"},
	}
	for _, testcase := range testcases {

		t.Run(testcase.testDescription, func(t *testing.T) {

			credentials, err := pool.Credentials(testcase.hyperscalerType, testcase.tenantName)
			actualError := ""
			if err != nil {
				actualError = err.Error()
				assert.Equal(t, testcase.expectedError, actualError)
			} else {
				assert.Equal(t, testcase.expectedCredentialName, credentials.Name)
				assert.Equal(t, testcase.hyperscalerType, credentials.HyperscalerType)
				assert.Equal(t, testcase.expectedCredentialName, string(credentials.CredentialData["credentials"]))
				assert.Equal(t, testcase.expectedError, actualError)
			}
		})
	}
}

func TestSecretsAccountPool_IsSecretInternal(t *testing.T) {
	t.Run("should return true if internal secret binding found", func(t *testing.T) {
		//given
		accPool := newTestAccountPoolWithSecretInternal()

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

func TestSecretsAccountPool_IsSecretDirty(t *testing.T) {
	t.Run("should return true if dirty secret binding found", func(t *testing.T) {
		//given
		accPool := newTestAccountPoolWithSecretDirty()

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

func TestSecretsAccountPool_IsSecretUsed(t *testing.T) {
	t.Run("should return true when secret binding is in use", func(t *testing.T) {
		//given
		accPool := newTestAccountPoolWithSingleShoot()

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

func TestSecretsAccountPool_MarkSecretAsDirty(t *testing.T) {
	t.Run("should mark secret binding as dirty", func(t *testing.T) {
		//given
		accPool, mockSecretBindings := newTestAccountPoolWithoutShoots()

		//when
		err := accPool.MarkSecretBindingAsDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := mockSecretBindings.Get("secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})
}

func newTestAccountPool() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret2", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret2"),
		},
	}
	secret3 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret3", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret3"),
		},
	}
	secret4 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret4", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret4"),
		},
	}
	secret5 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret5", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret5"),
		},
	}
	secret6 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret6", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret6"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1, secret2, secret3, secret4, secret5, secret6)
	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1, secretBinding2, secretBinding3, secretBinding4, secretBinding5, secretBinding6).
		CoreV1beta1().SecretBindings(testNamespace)

	pool := NewAccountPool(mockClient, gardenerFake, nil)
	return pool
}

func newTestAccountPoolWithSingleShoot() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots)
}

func newEmptyTestAccountPool() AccountPool {
	secret1 := &corev1.Secret{}
	secretBinding1 := &gardener_types.SecretBinding{}

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots)
}

func newTestAccountPoolWithSecretInternal() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots)
}

func newTestAccountPoolWithSecretDirty() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots)
}

func newTestAccountPoolWithShootsUsingSecret() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, shoot2, secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots)
}

func newTestAccountPoolWithoutShoots() (AccountPool, v1beta1.SecretBindingInterface) {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

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

	mockClient := fake.NewSimpleClientset(secret1)

	gardenerFake := gardener_fake.NewSimpleClientset(secretBinding1)
	mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	return NewAccountPool(mockClient, mockSecretBindings, mockShoots), mockSecretBindings
}
