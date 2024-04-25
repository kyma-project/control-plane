package aws

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkerConfig contains configuration settings for the worker nodes.
type WorkerConfig struct {
	metav1.TypeMeta `json:",inline"`

	// InstanceMetadataOptions contains configuration for controlling access to the metadata API.
	InstanceMetadataOptions *InstanceMetadataOptions `json:"instanceMetadataOptions,omitempty"`
}

// HTTPTokensValue is a constant for HTTPTokens values.
type HTTPTokensValue string

var (
	// HTTPTokensRequired is a constant for requiring the use of tokens to access IMDS. Effectively disables access via
	// the IMDSv1 endpoints.
	HTTPTokensRequired HTTPTokensValue = "required"
	// HTTPTokensOptional that makes the use of tokens for IMDS optional. Effectively allows access via both IMDSv1 and
	// IMDSv2 endpoints.
	HTTPTokensOptional HTTPTokensValue = "optional"
)

// InstanceMetadataOptions contains configuration for controlling access to the metadata API.
type InstanceMetadataOptions struct {
	// HTTPTokens enforces the use of metadata v2 API.
	HTTPTokens *HTTPTokensValue `json:"httpTokens,omitempty"`
	// HTTPPutResponseHopLimit is the response hop limit for instance metadata requests.
	// Valid values are between 1 and 64.
	HTTPPutResponseHopLimit *int64 `json:"httpPutResponseHopLimit,omitempty"`
}
