package openstack

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`
	// FloatingPoolName contains the FloatingPoolName name in which LoadBalancer FIPs should be created.
	FloatingPoolName string `json:"floatingPoolName"`
	// FloatingPoolSubnetName contains the name of a subnet in the Floating IP Pool where the router should be attached to.
	// +optional
	FloatingPoolSubnetName *string `json:"floatingPoolSubnetName,omitempty"`
	// Networks is the OpenStack specific network configuration
	Networks Networks `json:"networks"`
}

// Networks holds information about the Kubernetes and infrastructure networks.
type Networks struct {
	// Router indicates whether to use an existing router or create a new one.
	// +optional
	Router *Router `json:"router,omitempty"`
	// Worker is a CIDRs of a worker subnet (private) to create (used for the VMs).
	// Deprecated - use `workers` instead.
	Worker string `json:"worker"`
	// Workers is a CIDRs of a worker subnet (private) to create (used for the VMs).
	Workers string `json:"workers"`
}

// Router indicates whether to use an existing router or create a new one.
type Router struct {
	// ID is the router id of an existing OpenStack router.
	ID string `json:"id"`
}
