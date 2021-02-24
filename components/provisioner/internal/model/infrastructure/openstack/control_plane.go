package openstack

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ControlPlaneConfig struct {
	metav1.TypeMeta `json:",inline"`

	// CloudControllerManager contains configuration settings for the cloud-controller-manager.
	// +optional
	CloudControllerManager *CloudControllerManagerConfig `json:"cloudControllerManager,omitempty"`
	// LoadBalancerClasses available for a dedicated Shoot.
	// +optional
	LoadBalancerClasses []LoadBalancerClass `json:"loadBalancerClasses,omitempty"`
	// LoadBalancerProvider is the name of the load balancer provider in the OpenStack environment.
	LoadBalancerProvider string `json:"loadBalancerProvider"`
	// Zone is the OpenStack zone.
	// +optional
	// Deprecated: Don't use anymore. Will be removed in a future version.
	Zone *string `json:"zone,omitempty"`
}

// CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
type CloudControllerManagerConfig struct {
	// FeatureGates contains information about enabled feature gates.
	// +optional
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
}

// LoadBalancerClass defines a restricted network setting for generic LoadBalancer classes.
type LoadBalancerClass struct {
	// Name is the name of the LB class
	Name string `json:"name"`
	// FloatingSubnetID is the subnetwork ID of a dedicated subnet in floating network pool.
	// +optional
	FloatingSubnetID *string `json:"floatingSubnetID,omitempty"`
	// FloatingNetworkID is the network ID of the floating network pool.
	// +optional
	FloatingNetworkID *string `json:"floatingNetworkID,omitempty"`
	// SubnetID is the ID of a local subnet used for LoadBalancer provisioning. Only usable if no FloatingPool
	// configuration is done.
	// +optional
	SubnetID *string `json:"subnetID,omitempty"`
}
