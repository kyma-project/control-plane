package hyperscaler

//
//import (
//	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
//	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	corev1 "k8s.io/api/core/v1"
//	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/client-go/kubernetes/fake"
//
//	"testing"
//)
//
//
//func TestSharedPool_SharedCredentials(t *testing.T) {
//
//	for _, testCase := range []struct {
//		description    string
//		secrets        []runtime.Object
//		shoots         []runtime.Object
//		hyperscaler    Type
//		expectedSecret string
//	}{
//		{
//			description: "should get only Secrets with proper hyperscaler",
//			secrets: []runtime.Object{
//				newSecret("s1", "gcp", true),
//				newSecret("s2", "azure", true),
//				newSecret("s3", "aws", true),
//			},
//			shoots: []runtime.Object{
//				newShoot("sh1", "s1"),
//				newShoot("sh2", "s1"),
//				newShoot("sh3", "s1"),
//				newShoot("sh4", "s2"),
//			},
//			hyperscaler:    "gcp",
//			expectedSecret: "s1",
//		},
//		{
//			description: "should ignore not shared Secrets",
//			secrets: []runtime.Object{
//				newSecret("s1", "gcp", true),
//				newSecret("s2", "gcp", false),
//				newSecret("s3", "gcp", false),
//			},
//			shoots: []runtime.Object{
//				newShoot("sh1", "s1"),
//				newShoot("sh2", "s1"),
//				newShoot("sh3", "s1"),
//				newShoot("sh4", "s2"),
//			},
//			hyperscaler:    "gcp",
//			expectedSecret: "s1",
//		},
//		{
//			description: "should get least used Secret for GCP",
//			secrets: []runtime.Object{
//				newSecret("s1", "gcp", true),
//				newSecret("s2", "gcp", true),
//				newSecret("s3", "gcp", true),
//			},
//			shoots: []runtime.Object{
//				newShoot("sh1", "s1"),
//				newShoot("sh2", "s1"),
//				newShoot("sh3", "s1"),
//				newShoot("sh4", "s2"),
//				newShoot("sh5", "s2"),
//				newShoot("sh6", "s3"),
//			},
//			hyperscaler:    "gcp",
//			expectedSecret: "s3",
//		},
//		{
//			description: "should get least used Secret for Azure",
//			secrets: []runtime.Object{
//				newSecret("s1", "azure", true),
//				newSecret("s2", "azure", true),
//				newSecret("s3", "aws", true),
//			},
//			shoots: []runtime.Object{
//				newShoot("sh1", "s1"),
//				newShoot("sh2", "s1"),
//				newShoot("sh3", "s2"),
//			},
//			hyperscaler:    "azure",
//			expectedSecret: "s2",
//		},
//		{
//			description: "should get least used Secret for AWS",
//			secrets: []runtime.Object{
//				newSecret("s1", "aws", true),
//				newSecret("s2", "aws", true),
//			},
//			shoots: []runtime.Object{
//				newShoot("sh1", "s2"),
//			},
//			hyperscaler:    "aws",
//			expectedSecret: "s1",
//		},
//	} {
//		t.Run(testCase.description, func(t *testing.T) {
//			// given
//
//			mockClient := fake.NewSimpleClientset(testCase.secrets...)
//			mockSecrets := mockClient.CoreV1().Secrets(testNamespace)
//
//			gardenerFake := gardener_fake.NewSimpleClientset(testCase.shoots...)
//			mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
//
//			pool := NewSharedGardenerAccountPool(mockSecrets, mockShoots)
//
//			// when
//			credentials, err := pool.SharedCredentials(testCase.hyperscaler)
//			require.NoError(t, err)
//
//			// then
//			assert.Equal(t, testCase.expectedSecret, credentials.Name)
//		})
//	}
//}
//
//func TestSharedPool_SharedCredentials_Errors(t *testing.T) {
//
//	t.Run("should return error when no Secrets for hyperscaler found", func(t *testing.T) {
//		mockClient := fake.NewSimpleClientset(
//			newSecret("s1", "azure", true),
//			newSecret("s2", "gcp", false),
//		)
//		mockSecrets := mockClient.CoreV1().Secrets(testNamespace)
//
//		pool := NewSharedGardenerAccountPool(mockSecrets, nil)
//
//		// when
//		_, err := pool.SharedCredentials("gcp")
//
//		// then
//		require.Error(t, err)
//		assert.Contains(t, err.Error(), "no shared Secret found")
//	})
//}
//
//func newSecret(name, hyperscaler string, shared bool) *corev1.Secret {
//	secret := &corev1.Secret{
//		ObjectMeta: machineryv1.ObjectMeta{
//			Name: name, Namespace: testNamespace,
//			Labels: map[string]string{
//				"hyperscalerType": hyperscaler,
//			},
//		},
//		Data: map[string][]byte{
//			"credentials": []byte("secret1"),
//		},
//	}
//
//	if shared {
//		secret.Labels["shared"] = "true"
//	}
//
//	return secret
//}
//
//func newShoot(name, secret string) *gardener_types.Shoot {
//	return &gardener_types.Shoot{
//		ObjectMeta: machineryv1.ObjectMeta{
//			Name:      name,
//			Namespace: testNamespace,
//		},
//		Spec: gardener_types.ShootSpec{
//			SecretBindingName: secret,
//		},
//	}
//}
