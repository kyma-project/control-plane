package reconciler

// COPIED FROM RECONCILER keb/client.go - may be imported in the future

type Cluster struct {
	Cluster      string       `json:"runtimeID"`
	RuntimeInput RuntimeInput `json:"runtimeInput"`
	KymaConfig   KymaConfig   `json:"kymaConfig"`
	Metadata     Metadata     `json:"metadata"`
	Kubeconfig   string       `json:"kubeconfig"`
}

type RuntimeInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Configuration struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type Components struct {
	Component     string          `json:"component"`
	Namespace     string          `json:"namespace"`
	Configuration []Configuration `json:"configuration"`
}

type KymaConfig struct {
	Version        string       `json:"version"`
	Profile        string       `json:"profile"`
	Components     []Components `json:"components"`
	Administrators []string     `json:"administrators"`
}

type Metadata struct {
	GlobalAccountID string `json:"globalAccountID"`
	SubAccountID    string `json:"subAccountID"`
	ServiceID       string `json:"serviceID"`
	ServicePlanID   string `json:"servicePlanID"`
	ShootName       string `json:"shootName"`
	InstanceID      string `json:"instanceID"`
}

// reconciling statuses
const (
	ReconcilePendingStatus = "reconcile_pending"
	ReconcileFailedStatus = "reconcile_failed"
	ReconcilingStatus = "reconciling"
	ErrorStatus = "error"
	ReadyStatus = "ready"
)

type State struct {
	Cluster              string `json:"cluster"`
	ClusterVersion       int64  `json:"clusterVersion"`
	ConfigurationVersion int64  `json:"configurationVersion"`
	Status               string `json:"status"`
	StatusUrl            string `json:"statusUrl,omitempty"`
}

type StatusChange struct {
	Status   *string
	Duration string
}
