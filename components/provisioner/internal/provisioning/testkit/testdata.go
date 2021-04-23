package testkit

const (
	KymaSystemNamespace      = "kyma-system"
	KymaIntegrationNamespace = "kyma-integration"
	KymaVersion              = "1.8"
	KymaVersionWithoutTiller = "1.15"

	ClusterEssentialsComponent    = "cluster-essentials"
	CoreComponent                 = "core"
	RafterComponent               = "rafter"
	RafterSourceURL               = "github.com/kyma-project/kyma.git//resources/rafter"
	ApplicationConnectorComponent = "application-connector"
	ConnectivityProxyComponent    = "cloud-connectivity-proxy"

	CompassSystemNamespace = "compass-system"
	RuntimeAgentComponent  = "compass-runtime-agent"

	GardenerProject                            = "gardener-project"
	DefaultEnableKubernetesVersionAutoUpdate   = false
	DefaultEnableMachineImageVersionAutoUpdate = false
	ForceAllowPrivilegedContainers             = false
)
