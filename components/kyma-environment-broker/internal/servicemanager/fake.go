package servicemanager

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakeServiceManagerClient struct {
	offerings            []types.ServiceOffering
	plans                []types.ServicePlan
	provisioningResponse *ProvisionResponse
	provisionings        map[string]provisioningInfo

	unbindings map[string]InstanceKey
}

type provisioningInfo struct {
	BrokerID string
	Request  ProvisioningInput
}

type fakeServiceManagerClientFactory struct {
	cli *fakeServiceManagerClient
}

func NewFakeServiceManagerClientFactory(offerings []types.ServiceOffering, plans []types.ServicePlan) *fakeServiceManagerClientFactory {
	return &fakeServiceManagerClientFactory{
		cli: &fakeServiceManagerClient{
			offerings:     offerings,
			plans:         plans,
			provisionings: map[string]provisioningInfo{},
			unbindings:    map[string]InstanceKey{},
		},
	}
}

func (f *fakeServiceManagerClientFactory) ForCustomerCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (Client, error) {
	return f.cli, nil
}

func (f *fakeServiceManagerClientFactory) ProvideCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (*Credentials, error) {
	return reqCredentials, nil
}

func (f *fakeServiceManagerClient) ListOfferings() (*types.ServiceOfferings, error) {
	return nil, nil
}
func (f *fakeServiceManagerClient) ListOfferingsByName(name string) (*types.ServiceOfferings, error) {
	var result []types.ServiceOffering
	for _, off := range f.offerings {
		if off.Name == name {
			result = append(result, off)
		}
	}
	return &types.ServiceOfferings{
		ServiceOfferings: result,
	}, nil
}
func (f *fakeServiceManagerClient) ListPlansByName(planName, offeringID string) (*types.ServicePlans, error) {
	var result []types.ServicePlan
	for _, pl := range f.plans {
		if pl.Name == planName {
			result = append(result, pl)
		}
	}
	return &types.ServicePlans{
		ServicePlans: result,
	}, nil
}

func (f *fakeServiceManagerClient) Provision(brokerID string, request ProvisioningInput, acceptsIncomplete bool) (*ProvisionResponse, error) {
	f.provisionings[request.ID] = provisioningInfo{
		BrokerID: brokerID,
		Request:  request,
	}
	return f.provisioningResponse, nil
}

func (f *fakeServiceManagerClient) Deprovision(instanceKey InstanceKey, acceptsIncomplete bool) (*DeprovisionResponse, error) {
	return nil, nil
}

func (f *fakeServiceManagerClient) Bind(instanceKey InstanceKey, bindingID string, parameters interface{}, acceptsIncomplete bool) (*BindingResponse, error) {
	return nil, nil
}

func (f *fakeServiceManagerClient) Unbind(instanceKey InstanceKey, bindingID string, acceptsIncomplete bool) (*DeprovisionResponse, error) {
	f.unbindings[bindingID] = instanceKey
	return &DeprovisionResponse{
		OperationResponse: OperationResponse{},
		HTTPResponse:      HTTPResponse{StatusCode: http.StatusOK},
	}, nil
}

func (f *fakeServiceManagerClient) LastInstanceOperation(key InstanceKey, operationID string) (LastOperationResponse, error) {
	return LastOperationResponse{}, nil
}

// helper methods
func (f *fakeServiceManagerClientFactory) SynchronousProvisioning() {
	f.cli.provisioningResponse = &ProvisionResponse{
		ProvisionResponseBody: ProvisionResponseBody{
			OperationResponse: OperationResponse{
				OperationID: "",
			},
			Async:        false,
			DashboardURL: nil,
		},
		HTTPResponse: HTTPResponse{StatusCode: http.StatusCreated},
	}
}

// assertions
func (f *fakeServiceManagerClientFactory) AssertProvisionCalled(t *testing.T, instanceKey InstanceKey) {
	instance, exists := f.cli.provisionings[instanceKey.InstanceID]
	assert.True(t, exists, "provision endpoint was not called")

	assert.Equal(t, instance.BrokerID, instanceKey.BrokerID)
	assert.Equal(t, instance.Request.PlanID, instanceKey.PlanID)
	assert.Equal(t, instance.Request.ServiceID, instanceKey.ServiceID)
}

func (f *fakeServiceManagerClientFactory) AssertUnbindCalled(t *testing.T, key InstanceKey, bindingID string) {
	unbinding, exists := f.cli.unbindings[bindingID]
	assert.True(t, exists, "unbind endpoint was not called")

	assert.Equal(t, unbinding, key)
}
