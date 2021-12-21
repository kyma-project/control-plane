package gardener

import (
	"errors"
	"testing"

	gardener_Types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubernetesVersionProvider(t *testing.T) {
	t.Run("Get kubernetes version when succeeded to find Runtime", func(t *testing.T) {
		// given
		tenant := "tenant"
		shootClient := &mocks.ShootClient{}
		//List(ctx context.Context, opts v1.ListOptions) (*v1beta1.ShootList, error)
		shootClient.On("List", mock.Anything, metav1.ListOptions{LabelSelector: "account=" + tenant}).Return(&gardener_Types.ShootList{
			Items: []gardener_Types.Shoot{{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{runtimeIDAnnotation: "runtimeID"},
				},
				Spec: gardener_Types.ShootSpec{
					Kubernetes: gardener_Types.Kubernetes{
						Version: "1.21",
					},
				},
			},
			}}, nil)

		// when
		provider := NewKubernetesVersionProvider(shootClient)
		version, err := provider.Get("runtimeID", tenant)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.21", version)

	})

	t.Run("Return error when failed to find Runtime", func(t *testing.T) {
		// given
		tenant := "tenant"
		shootClient := &mocks.ShootClient{}
		//List(ctx context.Context, opts v1.ListOptions) (*v1beta1.ShootList, error)
		shootClient.On("List", mock.Anything, metav1.ListOptions{LabelSelector: "account=" + tenant}).Return(&gardener_Types.ShootList{
			Items: []gardener_Types.Shoot{{
				Spec: gardener_Types.ShootSpec{
					Kubernetes: gardener_Types.Kubernetes{
						Version: "1.21",
					},
				},
			},
			}}, nil)

		// when
		provider := NewKubernetesVersionProvider(shootClient)
		version, err := provider.Get("runtimeID", tenant)

		// then
		require.Error(t, err)
		assert.Equal(t, "", version)
	})

	t.Run("Return error when failed to get shoot", func(t *testing.T) {
		// given
		tenant := "tenant"
		shootClient := &mocks.ShootClient{}
		//List(ctx context.Context, opts v1.ListOptions) (*v1beta1.ShootList, error)
		shootClient.On("List", mock.Anything, metav1.ListOptions{LabelSelector: "account=" + tenant}).Return(nil, errors.New("oh, no!"))

		// when
		provider := NewKubernetesVersionProvider(shootClient)
		version, err := provider.Get("runtimeID", tenant)

		// then
		require.Error(t, err)
		assert.Equal(t, "", version)
	})
}
