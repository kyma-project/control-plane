package dbmodel

import (
	"time"
)

type RuntimeStateDTO struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`

	KymaConfig    string `json:"kymaConfig"`
	ClusterConfig string `json:"clusterConfig"`

	// these fields are also available in above configs
	// they are set separately to make fetching easier
	KymaVersion string
	K8SVersion  string
}
