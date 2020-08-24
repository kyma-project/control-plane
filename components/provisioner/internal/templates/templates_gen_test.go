package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateShootTemplate(t *testing.T) {

	expectedTemplate := `apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  creationTimestamp: null
  labels:
    account: ""
    subaccount: ""
  name: '{{ .ShootName }}'
  namespace: garden-{{ .ProjectName }}
spec:
  cloudProfileName: az
  kubernetes:
    allowPrivilegedContainers: true
    kubeAPIServer:
      enableBasicAuthentication: false
    version: 1.16.12
  maintenance:
    autoUpdate:
      kubernetesVersion: false
      machineImageVersion: false
  networking:
    nodes: 10.250.0.0/19
    type: calico
  provider:
    controlPlaneConfig:
      apiVersion: azure.provider.extensions.gardener.cloud/v1alpha1
      kind: ControlPlaneConfig
    infrastructureConfig:
      apiVersion: azure.provider.extensions.gardener.cloud/v1alpha1
      kind: InfrastructureConfig
      networks:
        vnet:
          cidr: 10.250.0.0/16
        workers: 10.250.0.0/16
      zoned: true
    type: azure
    workers:
    - machine:
        image:
          name: gardenlinux
          version: 27.1.0
        type: Standard_D8_v3
      maxSurge: 4
      maxUnavailable: 1
      maximum: 10
      minimum: 3
      name: cpu-worker-0
      volume:
        size: 50Gi
        type: Standard_LRS
      zones:
      - "1"
      - "2"
      - "3"
  purpose: development
  region: '{{ .Region }}'
  secretBindingName: '{{ .GardenerSecretName }}'
status:
  gardener:
    id: ""
    name: ""
    version: ""
  hibernated: false
  technicalID: ""
  uid: ""
`

	template, err := GenerateShootTemplate("azure")
	require.NoError(t, err)

	assert.Equal(t, expectedTemplate, string(template))
}

func TestGenerateShootTemplate_Error(t *testing.T) {

	t.Run("should return error if provider not supported", func(t *testing.T) {
		_, err := GenerateShootTemplate("not-a-provider")
		require.Error(t, err)
	})

}
