package hyperscaler

import (
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"testing"
)

func TestSharedPool_SharedCredentials(t *testing.T) {

	for _, testCase := range []struct {
		description    string
		secrets        []runtime.Object
		secretBindings []runtime.Object
		shoots         []runtime.Object
		hyperscaler    Type
		expectedSecret string
	}{
		{
			description: "should get only Secrets with proper hyperscaler",
			secrets: []runtime.Object{
				newSecret("s1"),
				newSecret("s2"),
				newSecret("s3"),
			},
			secretBindings: []runtime.Object{
				newSecretBinding("sb1", "s1", "gcp", true),
				newSecretBinding("sb2", "s2", "azure", true),
				newSecretBinding("sb3", "s3", "aws", true),
			},
			shoots: []runtime.Object{
				newShoot("sh1", "sb1"),
				newShoot("sh2", "sb1"),
				newShoot("sh3", "sb1"),
				newShoot("sh4", "sb2"),
			},
			hyperscaler:    "gcp",
			expectedSecret: "s1",
		},
		{
			description: "should ignore not shared Secrets",
			secrets: []runtime.Object{
				newSecret("s1"),
				newSecret("s2"),
				newSecret("s3"),
			},
			secretBindings: []runtime.Object{
				newSecretBinding("sb1", "s1", "gcp", true),
				newSecretBinding("sb2", "s2", "gcp", false),
				newSecretBinding("sb3", "s3", "gcp", false),
			},
			shoots: []runtime.Object{
				newShoot("sh1", "sb1"),
				newShoot("sh2", "sb1"),
				newShoot("sh3", "sb1"),
				newShoot("sh4", "sb2"),
			},
			hyperscaler:    "gcp",
			expectedSecret: "s1",
		},
		{
			description: "should get least used Secret for GCP",
			secrets: []runtime.Object{
				newSecret("s1"),
				newSecret("s2"),
				newSecret("s3"),
			},
			secretBindings: []runtime.Object{
				newSecretBinding("sb1", "s1", "gcp", true),
				newSecretBinding("sb2", "s2", "gcp", true),
				newSecretBinding("sb3", "s3", "gcp", true),
			},
			shoots: []runtime.Object{
				newShoot("sh1", "sb1"),
				newShoot("sh2", "sb1"),
				newShoot("sh3", "sb1"),
				newShoot("sh4", "sb2"),
				newShoot("sh5", "sb2"),
				newShoot("sh6", "sb3"),
			},
			hyperscaler:    "gcp",
			expectedSecret: "s3",
		},
		{
			description: "should get least used Secret for Azure",
			secrets: []runtime.Object{
				newSecret("s1"),
				newSecret("s2"),
				newSecret("s3"),
			},
			secretBindings: []runtime.Object{
				newSecretBinding("sb1", "s1", "azure", true),
				newSecretBinding("sb2", "s2", "azure", true),
				newSecretBinding("sb3", "s3", "aws", true),
			},
			shoots: []runtime.Object{
				newShoot("sh1", "sb1"),
				newShoot("sh2", "sb1"),
				newShoot("sh3", "sb2"),
			},
			hyperscaler:    "azure",
			expectedSecret: "s2",
		},
		{
			description: "should get least used Secret for AWS",
			secrets: []runtime.Object{
				newSecret("s1"),
				newSecret("s2"),
			},
			secretBindings: []runtime.Object{
				newSecretBinding("sb1", "s1", "aws", true),
				newSecretBinding("sb2", "s2", "aws", true),
			},
			shoots: []runtime.Object{
				newShoot("sh1", "sb2"),
			},
			hyperscaler:    "aws",
			expectedSecret: "s1",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			mockClient := fake.NewSimpleClientset(testCase.secrets...)
			gardenerFake := gardener_fake.NewSimpleClientset(append(testCase.shoots, testCase.secretBindings...)...)
			mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
			mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)

			pool := NewSharedGardenerAccountPool(mockClient, mockSecretBindings, mockShoots)

			// when
			credentials, err := pool.SharedCredentials(testCase.hyperscaler)
			require.NoError(t, err)

			// then
			assert.Equal(t, testCase.expectedSecret, credentials.Name)
		})
	}
}

func TestSharedPool_SharedCredentials_Errors(t *testing.T) {

	t.Run("should return error when no Secrets for hyperscaler found", func(t *testing.T) {
		// given
		mockClient := fake.NewSimpleClientset(
			newSecret("s1"),
			newSecret("s2"),
		)
		gardenerFake := gardener_fake.NewSimpleClientset(
			newSecretBinding("sb1", "s1", "azure", true),
			newSecretBinding("sb2", "s2", "gcp", false),
		)
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)

		pool := NewSharedGardenerAccountPool(mockClient, mockSecretBindings, nil)

		// when
		_, err := pool.SharedCredentials("gcp")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no shared secret binding found")
	})
}

func newSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: name, Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("secret1"),
		},
	}
}

func newSecretBinding(name, secretName, hyperscaler string, shared bool) *gardener_types.SecretBinding {
	secretBinding := &gardener_types.SecretBinding{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: name, Namespace: testNamespace,
			Labels: map[string]string{
				"hyperscalerType": hyperscaler,
			},
		},
		SecretRef: corev1.SecretReference{
			Name:      secretName,
			Namespace: testNamespace,
		},
	}

	if shared {
		secretBinding.Labels["shared"] = "true"
	}

	return secretBinding
}

func newShoot(name, secretBinding string) *gardener_types.Shoot {
	return &gardener_types.Shoot{
		ObjectMeta: machineryv1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: secretBinding,
		},
	}
}
