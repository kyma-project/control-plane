package testkit

const (
	KymaSystemNamespace      = "kyma-system"
	KymaIntegrationNamespace = "kyma-integration"
	KymaVersion              = "1.5"
	KymaVersionWithoutTiller = "1.15"

	ClusterEssentialsComponent    = "cluster-essentials"
	CoreComponent                 = "core"
	RafterComponent               = "rafter"
	ApplicationConnectorComponent = "application-connector"
	ConnectivityProxyComponent    = "cloud-connectivity-proxy"
	RafterSourceURL               = "github.com/kyma-project/kyma.git//resources/rafter"

	GardenerProject                            = "gardener-project"
	DefaultEnableKubernetesVersionAutoUpdate   = false
	DefaultEnableMachineImageVersionAutoUpdate = false
	ForceAllowPrivilegedContainers             = false
)
