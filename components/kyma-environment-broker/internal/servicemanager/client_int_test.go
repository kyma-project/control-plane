//go:build sm_integration
// +build sm_integration

package servicemanager

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
Those tests perform operation on the Service Manager using the client. Before running any test set the following envs:
 - SM_USERNAME
 - SM_PASSWORD
 - SM_URL
*/

// Running: go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_ListOfferings
func TestClient_ListOfferings(t *testing.T) {
	// given
	client := newClient(t)

	// when
	offerings, err := client.ListOfferings()

	// then
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	if offerings == nil {
		fmt.Println("offerings nil")
	}
	for _, offering := range offerings.ServiceOfferings {
		fmt.Printf("Name: %s, %s, %s \n", offering.Name, offering.ID, offering.BrokerName)
	}
}

// Running: go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_ListOfferingsByName
// Optional environment variable with the offering name, for example:
// export OFFERING_NAME=xsuaa
func TestClient_ListOfferingsByName(t *testing.T) {
	// given
	client := newClient(t)
	offeringName := os.Getenv("OFFERING_NAME")
	if offeringName == "" {
		offeringName = "xsuaa"
	}

	// when
	offerings, err := client.ListOfferingsByName(offeringName)

	// then
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	if offerings == nil {
		fmt.Println("offerings nil")
	}
	for _, offering := range offerings.ServiceOfferings {
		fmt.Printf(offering.TableData().String())
	}
}

// Running: go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_ListPlansByName
// Optional environment variables with the offering and plan names, for example:
// export OFFERING_NAME=xsuaa
// export PLAN_NAME=application
func TestClient_ListPlansByName(t *testing.T) {
	// given
	client := newClient(t)
	offeringName := os.Getenv("OFFERING_NAME")
	if offeringName == "" {
		offeringName = "xsuaa"
	}
	planName := os.Getenv("PLAN_NAME")
	if planName == "" {
		planName = "application"
	}
	offerings, err := client.ListOfferingsByName(offeringName)
	require.NoError(t, err)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]

	// when
	plans, err := client.ListPlansByName(planName, offering.ID)

	// then
	require.NoError(t, err)
	fmt.Println(plans.TableData().String())
}

// Running: go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_Provision
// Optional environment variable with the accepts incomplete flag, for example:
// export ACCEPTS_INCOMPLETE=false
// export OFFERING_NAME=xsuaa
// export PLAN_NAME=application
func TestClient_Provision(t *testing.T) {
	client := newClient(t)
	offeringName := os.Getenv("OFFERING_NAME")
	if offeringName == "" {
		offeringName = "xsuaa"
	}
	planName := os.Getenv("PLAN_NAME")
	if planName == "" {
		planName = "application"
	}
	offerings, err := client.ListOfferingsByName(offeringName)
	require.NoError(t, err)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]
	plans, err := client.ListPlansByName(planName, offering.ID)
	require.NoError(t, err)
	assert.Len(t, plans.ServicePlans, 1)
	plan := plans.ServicePlans[0]
	acceptsIncomplete := strings.ToLower(os.Getenv("ACCEPTS_INCOMPLETE")) == "true"

	// when
	input := ProvisioningInput{
		ID: uuid.New().String(),
		ProvisionRequest: ProvisionRequest{
			ServiceID:  offering.CatalogID,
			PlanID:     plan.CatalogID,
			Parameters: nil,
			Context: map[string]interface{}{
				"platform": "kubernetes",
			},
			OrganizationGUID: uuid.New().String(),
			SpaceGUID:        uuid.New().String(),
		},
	}
	fmt.Printf("Provisioning Instance ID=%s accepts_incomplete=%v\n", input.ID, acceptsIncomplete)
	resp, err := client.Provision(offering.BrokerID, input, acceptsIncomplete)

	fmt.Println(err)
	fmt.Printf("Response %+v\n", resp)
	fmt.Printf("export INSTANCE_ID=%s\n", input.ID)
}

// Running:
//
// export INSTANCE_ID=abcd-0001
// go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_Deprovision
//
// Optional environment variables, for example:
// export ACCEPTS_INCOMPLETE=false
func TestClient_Deprovision(t *testing.T) {
	client := newClient(t)
	offerings, err := client.ListOfferingsByName("xsuaa")
	require.NoError(t, err)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]
	plans, err := client.ListPlansByName("application", offering.ID)
	require.NoError(t, err)
	assert.Len(t, plans.ServicePlans, 1)

	plan := plans.ServicePlans[0]
	instanceID := os.Getenv("INSTANCE_ID")
	acceptsIncomplete := strings.ToLower(os.Getenv("ACCEPTS_INCOMPLETE")) == "true"

	// when
	resp, err := client.Deprovision(InstanceKey{
		BrokerID:   offering.BrokerID,
		InstanceID: instanceID,
		ServiceID:  offering.CatalogID,
		PlanID:     plan.CatalogID,
	}, acceptsIncomplete)

	fmt.Println(err)
	fmt.Printf("Response %+v\n", resp)
}

// Running:
//
// export INSTANCE_ID=abcd-0001
// go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_Bind
//
// Optional environment variables, for example:
// export ACCEPTS_INCOMPLETE=false
func TestClient_Bind(t *testing.T) {
	client := newClient(t)
	offerings, err := client.ListOfferingsByName("xsuaa")
	require.NoError(t, err)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]
	plans, err := client.ListPlansByName("application", offering.ID)
	require.NoError(t, err)
	assert.Len(t, plans.ServicePlans, 1)

	plan := plans.ServicePlans[0]
	instanceID := os.Getenv("INSTANCE_ID")
	bindingID := uuid.New().String()
	acceptsIncomplete := strings.ToLower(os.Getenv("ACCEPTS_INCOMPLETE")) == "true"

	// when
	resp, err := client.Bind(InstanceKey{
		BrokerID:   offering.BrokerID,
		InstanceID: instanceID,
		ServiceID:  offering.CatalogID,
		PlanID:     plan.CatalogID,
	}, bindingID, nil, acceptsIncomplete)

	fmt.Println(err)
	fmt.Printf("Response %+v\n", resp)
	fmt.Printf("export BINDING_ID=%s\n", bindingID)
}

func TestClient_LastInstanceOperation(t *testing.T) {
	client := newClient(t)
	instanceID := os.Getenv("INSTANCE_ID")
	offeringName := os.Getenv("OFFERING_NAME")
	if offeringName == "" {
		offeringName = "xsuaa"
	}
	planName := os.Getenv("PLAN_NAME")
	if planName == "" {
		planName = "application"
	}
	offerings, err := client.ListOfferingsByName(offeringName)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]
	require.NoError(t, err)
	plans, err := client.ListPlansByName(planName, offering.ID)
	require.NoError(t, err)
	assert.Len(t, plans.ServicePlans, 1)

	plan := plans.ServicePlans[0]

	fmt.Println(instanceID)
	fmt.Println(offering.BrokerID)
	resp, err := client.LastInstanceOperation(InstanceKey{
		BrokerID:   offering.BrokerID,
		InstanceID: instanceID,
		ServiceID:  offering.CatalogID,
		PlanID:     plan.CatalogID,
	}, "")
	fmt.Printf("%+v", resp)
}

// Running:
//
// export INSTANCE_ID=abcd-0001
// export BINDING_ID=abcd-0001
// go test -v -tags=sm_integration ./internal/servicemanager/... -run TestClient_Unbind
//
// Optional environment variables:
// export ACCEPTS_INCOMPLETE=false
func TestClient_Unbind(t *testing.T) {
	client := newClient(t)
	offerings, err := client.ListOfferingsByName("xsuaa")
	require.NoError(t, err)
	require.Len(t, offerings.ServiceOfferings, 1)
	offering := offerings.ServiceOfferings[0]
	plans, err := client.ListPlansByName("application", offering.ID)
	require.NoError(t, err)
	assert.Len(t, plans.ServicePlans, 1)

	plan := plans.ServicePlans[0]
	instanceID := os.Getenv("INSTANCE_ID")
	bindingID := os.Getenv("BINDING_ID")
	acceptsIncomplete := strings.ToLower(os.Getenv("ACCEPTS_INCOMPLETE")) == "true"

	// when
	resp, err := client.Unbind(InstanceKey{
		BrokerID:   offering.BrokerID,
		InstanceID: instanceID,
		ServiceID:  offering.CatalogID,
		PlanID:     plan.CatalogID,
	}, bindingID, acceptsIncomplete)

	fmt.Println(err)
	fmt.Printf("Response %+v\n", resp)
}

func newClient(t *testing.T) Client {
	t.Helper()

	username := os.Getenv("SM_USERNAME")
	password := os.Getenv("SM_PASSWORD")
	url := os.Getenv("SM_URL")

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      url,
	})

	return client
}
