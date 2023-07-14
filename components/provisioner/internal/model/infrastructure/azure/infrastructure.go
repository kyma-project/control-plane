package azure

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// This types are copied from https://github.com/gardener/gardener-extensions/blob/master/controllers/provider-azure/pkg/apis/azure/types_infrastructure.go as it does not contain json tags

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta
	// ResourceGroup is azure resource group
	ResourceGroup *ResourceGroup `json:"resourceGroup,omitempty"`
	// Networks is the network configuration (VNets, subnets, etc.)
	Networks NetworkConfig `json:"networks"`
	// Zoned indicates whether the cluster uses zones
	Zoned bool `json:"zoned"`
}

// ResourceGroup is azure resource group
type ResourceGroup struct {
	// Name is the name of the resource group
	Name string `json:"name"`
}

// NetworkConfig holds information about the Kubernetes and infrastructure networks.
type NetworkConfig struct {
	// VNet indicates whether to use an existing VNet or create a new one.
	VNet VNet `json:"vnet"`
	// Workers is the worker subnet range to create (used for the VMs).
	// +optional
	Workers *string `json:"workers,omitempty"`
	// ServiceEndpoints is a list of Azure ServiceEndpoints which should be associated with the worker subnet.
	ServiceEndpoints []string    `json:"serviceEndpoints,omitempty"`
	NatGateway       *NatGateway `json:"natGateway,omitempty"`
	Zones            []Zone      `json:"zones,omitempty"`
}

// VNet contains information about the VNet and some related resources.
type VNet struct {
	// Name is the VNet name.
	Name *string `json:"name,omitempty"`
	// ResourceGroup is the resource group where the existing vNet belongs to.
	ResourceGroup *string `json:"resourceGroup,omitempty"`
	// CIDR is the VNet CIDR
	CIDR *string `json:"cidr,omitempty"`
}

// VNetStatus contains the VNet name.
type VNetStatus struct {
	// Name is the VNet name.
	Name string `json:"name"`
	// ResourceGroup is the resource group where the existing vNet belongs to.
	ResourceGroup *string `json:"resourceGroup,omitempty"`
}

type NatGateway struct {
	// Enabled is an indicator if NAT gateway should be deployed.
	Enabled bool `json:"enabled"`
	// IdleConnectionTimeoutMinutes specifies the idle connection timeout limit for NAT gateway in minutes.
	IdleConnectionTimeoutMinutes int `json:"idleConnectionTimeoutMinutes"`
	// Zone specifies the zone in which the NAT gateway should be deployed to.
	Zone int `json:"zone,omitempty"`
	// IPAddresses is a list of ip addresses which should be assigned to the NAT gateway.
	IPAddresses []PublicIPReference `json:"ipAddresses,omitempty"`
}

// PublicIPReference contains information about a public ip.
type PublicIPReference struct {
	// Name is the name of the public ip.
	Name string `json:"name"`
	// ResourceGroup is the name of the resource group where the public ip is assigned to.
	ResourceGroup string `json:"resourceGroup"`
	// Zone is the zone in which the public ip is deployed to.
	Zone int32 `json:"zone,omitempty"`
}

type Zone struct {
	// Name is the name of the zone and should match with the name the infrastructure provider is using for the zone.
	Name int `json:"name"`
	// CIDR is the CIDR range used for the zone's subnet.
	CIDR string `json:"cidr"`
	// ServiceEndpoints is a list of Azure ServiceEndpoints which should be associated with the zone's subnet.
	// +optional
	ServiceEndpoints []string `json:"serviceEndpoints,omitempty"`
	// NatGateway contains the configuration for the NatGateway associated with this subnet.
	// +optional
	NatGateway *NatGateway `json:"natGateway,omitempty"`
}
