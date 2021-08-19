package kubeconfig

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/stretchr/testify/require"
)

const (
	globalAccountID = "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e"
	runtimeID       = "f7d634ae-4ce2-4916-be64-b6fb493155df"

	issuerURL = "https://example.com"
	clientID  = "c1id"
)

func TestBuilder_Build(t *testing.T) {
	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{
			RuntimeConfiguration: &schema.RuntimeConfig{
				Kubeconfig: skrKubeconfig(),
				ClusterConfig: &schema.GardenerConfig{
					OidcConfig: &schema.OIDCConfig{
						ClientID:       clientID,
						GroupsClaim:    "gclaim",
						IssuerURL:      issuerURL,
						SigningAlgs:    nil,
						UsernameClaim:  "uclaim",
						UsernamePrefix: "-",
					},
				},
			},
		}, nil)
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfig())
	})

	t.Run("provisioner client returned error", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{}, fmt.Errorf("cannot return kubeconfig"))
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient)
		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		_, err := builder.Build(instance)

		//then
		require.Error(t, err)
		require.Contains(t, err.Error(), "while fetching runtime status from provisioner: cannot return kubeconfig")
	})

	t.Run("provisioner client returned wrong kubeconfig", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{
			RuntimeConfiguration: &schema.RuntimeConfig{
				Kubeconfig: skrWrongKubeconfig(),
			},
		}, nil)
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient)
		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		_, err := builder.Build(instance)

		//then
		require.Error(t, err)
		require.Contains(t, err.Error(), "while validation kubeconfig fetched by provisioner")
	})
}

func skrKubeconfig() *string {
	kc := `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9-token
users:
- name: shoot--kyma-dev--ac0d8d9-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw
`
	return &kc
}

func newKubeconfig() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9
users:
- name: shoot--kyma-dev--ac0d8d9
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - oidc-login
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      command: kubectl
`, issuerURL, clientID,
	)
}

func skrWrongKubeconfig() *string {
	kc := `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9-token
users:
- name: shoot--kyma-dev--ac0d8d9-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw
`
	return &kc
}
