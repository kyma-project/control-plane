package hyperscaler

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
			"failed to find unassigned secret for hyperscalerType: azure"},

		{"No Available credential for tenant6, GCP returns error - ignore secret with label shared=true",
			"tenant6", GCP, "",
			"failed to find unassigned secret for hyperscalerType: gcp"},
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

func TestSecretsAccountPool_IsSecretDirty(t *testing.T) {
	t.Run("should return true if dirty secret found", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithSecretDirty()

		//when
		isdirty, err := accPool.IsSecretDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.True(t, isdirty)
	})

	t.Run("should return false if dirty secret not found", func(t *testing.T) {
		//given
		accPool := newTestAccountPool()

		//when
		isdirty, err := accPool.IsSecretDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, isdirty)
	})
}

func TestSecretsAccountPool_IsSecretUsed(t *testing.T) {
	t.Run("should return true when secret is in use", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithSingleShoot()

		//when
		used, err := accPool.IsSecretUsed("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.True(t, used)
	})

	t.Run("should return false when secret is not in use", func(t *testing.T) {
		//given
		accPool, _ := newTestAccountPoolWithoutShoots()

		//when
		used, err := accPool.IsSecretUsed("azure", "tenant1")

		//then
		require.NoError(t, err)
		assert.False(t, used)
	})
}

func TestSecretsAccountPool_MarkSecretAsDirty(t *testing.T) {
	t.Run("should mark secret as dirty", func(t *testing.T) {
		//given
		accPool, mockSecrets := newTestAccountPoolWithoutShoots()

		//when
		err := accPool.MarkSecretAsDirty("azure", "tenant1")

		//then
		require.NoError(t, err)
		secret, err := mockSecrets.Get("secret1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secret.Labels["dirty"], "true")
	})
}

func newTestAccountPool() AccountPool {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "gcp",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret2", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret2"),
		},
	}
	secret3 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret3", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant2",
				"hyperscalerType": "gcp",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret3"),
		},
	}
	secret4 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret4", Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "gcp",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret4"),
		},
	}
	secret5 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret5", Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "aws",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret5"),
		},
	}
	secret6 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret6", Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": "gcp",
				"shared":          "true",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret6"),
		},
	}

	mockClient := fake.NewSimpleClientset(secret1, secret2, secret3, secret4, secret5, secret6)
	mockSecrets := mockClient.CoreV1().Secrets(testNamespace)
	pool := NewAccountPool(mockSecrets, nil)
	return pool
}

func newTestAccountPoolWithSingleShoot() (AccountPool, v1.SecretInterface) {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secret1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	mockClient := fake.NewSimpleClientset(secret1)
	mockSecrets := mockClient.CoreV1().Secrets(testNamespace)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	pool := NewAccountPool(mockSecrets, mockShoots)

	return pool, mockSecrets
}

func newTestAccountPoolWithSecretDirty() (AccountPool, v1.SecretInterface) {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
				"dirty":           "true",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secret1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	mockClient := fake.NewSimpleClientset(secret1)
	mockSecrets := mockClient.CoreV1().Secrets(testNamespace)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	pool := NewAccountPool(mockSecrets, mockShoots)
	return pool, mockSecrets
}

func newTestAccountPoolWithShootsUsingSecret() (AccountPool, v1.SecretInterface) {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

	shoot1 := &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      "shoot1",
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: "secret1",
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
			SecretBindingName: "secret1",
		},
		Status: gardener_types.ShootStatus{
			LastOperation: &gardener_types.LastOperation{
				State: gardener_types.LastOperationStateSucceeded,
				Type:  gardener_types.LastOperationTypeReconcile,
			},
		},
	}

	mockClient := fake.NewSimpleClientset(secret1)
	mockSecrets := mockClient.CoreV1().Secrets(testNamespace)

	gardenerFake := gardener_fake.NewSimpleClientset(shoot1, shoot2)
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	pool := NewAccountPool(mockSecrets, mockShoots)
	return pool, mockSecrets
}

func newTestAccountPoolWithoutShoots() (AccountPool, v1.SecretInterface) {
	secret1 := &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "secret1", Namespace: testNamespace,
			Labels: map[string]string{
				"tenantName":      "tenant1",
				"hyperscalerType": "azure",
			},
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}

	mockClient := fake.NewSimpleClientset(secret1)
	mockSecrets := mockClient.CoreV1().Secrets(testNamespace)

	gardenerFake := gardener_fake.NewSimpleClientset()
	mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

	pool := NewAccountPool(mockSecrets, mockShoots)

	return pool, mockSecrets
}
