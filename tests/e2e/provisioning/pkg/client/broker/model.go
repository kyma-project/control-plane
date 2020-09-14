package broker

type (
	UpgradeRuntimeRequest struct {
		Targets Target `json:"targets"`
	}

	Target struct {
		Include []RuntimeTarget `json:"include"`
	}

	RuntimeTarget struct {
		RuntimeID string `json:"runtimeId,omitempty"`
	}

	UpgradeRuntimeResponse struct {
		OrchestrationID string `json:"orchestration_id"`
	}
)

type Runtime struct {
	RuntimeID         string `json:"runtimeId"`
	ServiceInstanceID string `json:"serviceInstanceId"`
}

type OrchestrationResponse struct {
	OrchestrationID string
	State           string
}
