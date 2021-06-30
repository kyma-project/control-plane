package servicemanager

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	FakeEmsServiceID = "fake-ems-svc-id"
)

type passthroughServiceManagerClientFactory struct {
	cli Client
}

func NewPassthroughServiceManagerClientFactory(cli Client) *passthroughServiceManagerClientFactory {
	return &passthroughServiceManagerClientFactory{
		cli: cli,
	}
}

func (f *passthroughServiceManagerClientFactory) ForCredentials(credentials *Credentials) Client {
	return f.cli
}

func (f *passthroughServiceManagerClientFactory) ForCustomerCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (Client, error) {
	return f.cli, nil
}

func (f *passthroughServiceManagerClientFactory) ProvideCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (*Credentials, error) {
	return reqCredentials, nil
}

type fakeServiceManagerClient struct {
	offerings            []types.ServiceOffering
	plans                []types.ServicePlan
	provisioningResponse *ProvisionResponse
	provisionings        map[string]provisioningInfo
	bindings             map[string]InstanceKey
	unbindings           map[string]InstanceKey
	deprovisions         map[string]InstanceKey
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
			bindings:      map[string]InstanceKey{},
			unbindings:    map[string]InstanceKey{},
			deprovisions:  map[string]InstanceKey{},
		},
	}
}

func (f *fakeServiceManagerClientFactory) ForCredentials(credentials *Credentials) Client {
	return f.cli
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
	f.deprovisions[instanceKey.InstanceID] = instanceKey
	return &DeprovisionResponse{
		OperationResponse: OperationResponse{},
		HTTPResponse:      HTTPResponse{StatusCode: http.StatusOK},
	}, nil
}

func (f *fakeServiceManagerClient) Bind(instanceKey InstanceKey, bindingID string, parameters interface{}, acceptsIncomplete bool) (*BindingResponse, error) {
	f.bindings[bindingID] = instanceKey

	return &BindingResponse{
		Binding: f.fixEmsBinding(),
	}, nil
}

func (f *fakeServiceManagerClient) Unbind(instanceKey InstanceKey, bindingID string, acceptsIncomplete bool) (*DeprovisionResponse, error) {
	f.unbindings[bindingID] = instanceKey
	return &DeprovisionResponse{
		OperationResponse: OperationResponse{},
		HTTPResponse:      HTTPResponse{StatusCode: http.StatusOK},
	}, nil
}
func (f *fakeServiceManagerClient) GetBinding(instanceKey InstanceKey, bindingID string) (*types.ServiceBinding, error) {
	return nil, nil
}

func (f *fakeServiceManagerClient) LastInstanceOperation(key InstanceKey, operationID string) (LastOperationResponse, error) {
	return LastOperationResponse{
		State: Succeeded,
	}, nil
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

func (f *fakeServiceManagerClientFactory) AssertDeprovisionCalled(t *testing.T, key InstanceKey) {
	deprovision, exists := f.cli.deprovisions[key.InstanceID]
	assert.True(t, exists, "deprovision endpoint was not called")

	assert.Equal(t, deprovision, key)
}

func (f *fakeServiceManagerClient) fixEmsBinding() Binding {
	return Binding{Credentials: map[string]interface{}{
		"clientid":                             "connectivity-oa2-clientid",            // For connectivity
		"clientsecret":                         "connectivity-oa2-clientsecret",        // For connectivity
		"subaccount_id":                        "subaccount_id",                        // For connectivity
		"subaccount_subdomain":                 "subaccount_subdomain",                 // For connectivity
		"token_service_domain":                 "token_service_domain",                 // For connectivity
		"token_service_url":                    "token_service_url",                    // For connectivity
		"token_service_url_pattern":            "toke_service_url_pattern",             // For connectivity
		"token_service_url_pattern_tenant_key": "token_service_url_pattern_tenant_key", // For connectivity
		"management": []interface{}{
			map[string]interface{}{
				"oa2": map[string]interface{}{
					"clientid":      "management-oa2-clientid",
					"clientsecret":  "management-oa2-clientsecret",
					"granttype":     "management-oa2-granttype",
					"tokenendpoint": "management-oa2-tokenendpoint",
				},
				"uri": "https://management-uri",
			},
		},
		// For connectivity
		"connectivity_service": map[string]interface{}{
			"CAs_path":         "...",
			"CAs_signing_path": "...",
			"api_path":         "...",
			"tunnel_path":      "...",
			"url":              "...",
		},
		"messaging": []interface{}{
			map[string]interface{}{
				"broker": map[string]interface{}{
					"type": "sapmgw",
				},
				"oa2": map[string]interface{}{
					"clientid":      "messaging-amqp10ws-oa2-clientid",
					"clientsecret":  "messaging-amqp10ws-oa2-clientsecret",
					"granttype":     "messaging-amqp10ws-oa2-granttype",
					"tokenendpoint": "https://messaging-amqp10ws-oa2-tokenendpoint",
				},
				"protocol": []interface{}{
					"amqp10ws",
				},
				"uri": "wss://messaging-amqp10ws-oa2-uri",
			},
			map[string]interface{}{
				"broker": map[string]interface{}{
					"type": "sapmgw",
				},
				"oa2": map[string]interface{}{
					"clientid":      "messaging-mqtt311ws-oa2-clientid",
					"clientsecret":  "messaging-mqtt311ws-oa2-clientsecret",
					"granttype":     "messaging-mqtt311ws-oa2-granttype",
					"tokenendpoint": "https://messaging-mqtt311ws-oa2-tokenendpoint",
				},
				"protocol": []interface{}{
					"mqtt311ws",
				},
				"uri": "wss://messaging-mqtt311ws-oa2-uri",
			},
			map[string]interface{}{
				"broker": map[string]interface{}{
					"type": "saprestmgw",
				},
				"oa2": map[string]interface{}{
					"clientid":      "messaging-httprest-oa2-clientid",
					"clientsecret":  "messaging-httprest-oa2-clientsecret",
					"granttype":     "messaging-httprest-oa2-granttype",
					"tokenendpoint": "https://messaging-httprest-oa2-tokenendpoint",
				},
				"protocol": []interface{}{
					"httprest",
				},
				"uri": "https://messaging-httprest-oa2-uri",
			},
		},
		"namespace":         "kyma-namespace",
		"serviceinstanceid": "serviceinstanceid",
		"xsappname":         "xsappname",
	}}
}
