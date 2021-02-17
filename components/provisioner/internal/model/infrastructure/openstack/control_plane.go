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
