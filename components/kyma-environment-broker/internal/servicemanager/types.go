package servicemanager

import (
	"net/http"

	"github.com/Peripli/service-manager-cli/pkg/types"
)

type (
	Client interface {
		ListOfferings() (*types.ServiceOfferings, error)
		ListOfferingsByName(name string) (*types.ServiceOfferings, error)
		ListPlansByName(planName, offeringID string) (*types.ServicePlans, error)
		Provision(brokerID string, request ProvisioningInput, acceptsIncomplete bool) (*ProvisionResponse, error)
		Deprovision(instanceKey InstanceKey, acceptsIncomplete bool) (*DeprovisionResponse, error)
		Bind(instanceKey InstanceKey, bindingID string, parameters interface{}, acceptsIncomplete bool) (*BindingResponse, error)
		Unbind(instanceKey InstanceKey, bindingID string, acceptsIncomplete bool) (*DeprovisionResponse, error)
		LastInstanceOperation(key InstanceKey, operationID string) (LastOperationResponse, error)
	}

	// InstanceKey contains all identifiers which allows us to perform all actions on an instance:
	// - bind
	// - unbind
	// - deprovision
	InstanceKey struct {
		BrokerID   string
		InstanceID string
		ServiceID  string
		PlanID     string
	}
)

type ProvisionRequest struct {
	ServiceID  string                 `json:"service_id"`
	PlanID     string                 `json:"plan_id"`
	Parameters interface{}            `json:"parameters,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`

	OrganizationGUID string `json:"organization_guid"`
	SpaceGUID        string `json:"space_guid"`
}

// ProvisioningInput aggregates provisioning parameters
type ProvisioningInput struct {
	ProvisionRequest
	ID string
}

type OperationResponse struct {
	OperationID string `json:"operation"`
}

type ProvisionResponseBody struct {
	OperationResponse `json:""`
	Async             bool    `json:"async"`
	DashboardURL      *string `json:"dashboard_url,omitempty"`
}

type HTTPResponse struct {
	StatusCode int
}

type ProvisionResponse struct {
	ProvisionResponseBody
	HTTPResponse
}

type DeprovisionResponse struct {
	OperationResponse
	HTTPResponse
}

type UnbindResponse struct {
	OperationResponse
	HTTPResponse
}

type Binding struct {
	Credentials map[string]interface{} `json:"credentials"`
}

type BindingResponse struct {
	Binding
	HTTPResponse
}

type LastOperationResponse struct {
	State       LastOperationState `json:"state"`
	Description string             `json:"description"`
}

type LastOperationState string

const (
	InProgress LastOperationState = "in progress"
	Succeeded  LastOperationState = "succeeded"
	Failed     LastOperationState = "failed"
)

func (pr *HTTPResponse) IsDone() bool {
	if pr == nil {
		return false
	}
	return pr.StatusCode == http.StatusOK || pr.StatusCode == http.StatusCreated
}

func (pr *HTTPResponse) IsInProgress() bool {
	if pr == nil {
		return false
	}
	return pr.StatusCode == http.StatusAccepted
}

func (pr *OperationResponse) GetOperationID() string {
	if pr == nil {
		return ""
	}
	return pr.OperationID
}
