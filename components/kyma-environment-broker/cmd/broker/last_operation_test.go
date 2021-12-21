package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestLastOperationWithoutOperationIDHappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	//when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation",iid), "")

	//then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLastOperationWithOperationIDHappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	//when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", iid, opID), "")

	//then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNotExistingInstance(t *testing.T){
	//given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	//when
	resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), "")

	//then
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Contains(t, errResponse.Description, fmt.Sprintf("instance operation with instance_id %s not found", iid))
}

func TestNotExistingOperation(t *testing.T){
	//given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()
	oid := uuid.New().String()

	//when
	resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", iid, oid), "")

	//then
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Contains(t, errResponse.Description, fmt.Sprintf("instance operation with id %s not found", oid))
}

func TestLastOperationWithOperationIDAndWithoutInstanceID(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	oid := uuid.New().String()
	resp2 := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", oid, opID), "")
	//assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	fmt.Printf("%v", resp2.StatusCode)
}