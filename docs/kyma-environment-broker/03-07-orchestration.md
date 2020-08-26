---
title: Orchestration
type: Details
---

## Handlers 

Orchestration handlers allows to fetch orchestration status and to run upgrade of kyma or cluster.

The handlers are as follows:

- `GET /orchestrations/{orchestration_id}`

**Responds** with the orchestration object using struct: 

```
type Orchestration struct {
	OrchestrationID   string
	State             string
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Parameters        Parameters
	RuntimeOperations RuntimeOperation
}
```
```
type Parameters struct {
	Targets  internal.TargetSpec   `json:"targets"`
	Strategy internal.StrategySpec `json:"strategy,omitempty"`
}

type UpgradeOrchestrationResponseDTO struct {
	OperationID string `json:"operation_id"`
}
```

- `POST /upgrade/kyma`

With the following **body**:

```
type UpgradeOrchestrationDTO struct {
	Targets  internal.TargetSpec   `json:"targets"`
	Strategy internal.StrategySpec `json:"strategy,omitempty"`
}
```
```
type StrategySpec struct {
	Type     StrategyType         `json:"type"`
	Schedule ScheduleType         `json:"schedule,omitempty"`
	Parallel ParallelStrategySpec `json:"parallel,omitempty"`
}
type TargetSpec struct {
	Include []RuntimeTarget `json:"include"`
	Exclude []RuntimeTarget `json:"exclude,omitempty"`
}
type RuntimeTarget struct {
	// Valid values: "all"
	Target string `json:"target,omitempty"`
	GlobalAccount string `json:"globalAccount,omitempty"`
	SubAccount string `json:"subAccount,omitempty"`
	Region string `json:"region,omitempty"`
	RuntimeID string `json:"runtimeId,omitempty"`
}
```

**Responds** with newly created orchestrationID using the following struct:

```
type UpgradeOrchestrationResponseDTO struct {
	OrchestrationID string `json:"orchestration_id"`
}
```