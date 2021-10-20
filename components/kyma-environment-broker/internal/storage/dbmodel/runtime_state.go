package dbmodel

import (
	"time"
)

type RuntimeStateDTO struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`

	KymaConfig    string `json:"kymaConfig"`
	ClusterConfig string `json:"clusterConfig"`
	ClusterSetup  string `json:"clusterSetup,omitempty"`

	// these fields are also available in above configs
	// they are set separately to make fetching easier
	KymaVersion string `json:"kyma_version"`
	K8SVersion  string `json:"k8s_version"`
}
