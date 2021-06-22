package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestUpdateNotExistingInstance(t *testing.T) {
	suite := NewUpdateSuite(t)
	defer suite.TearDown()

	resp := suite.CallAPI("PATCH", "oauth/v2/service_instances/not-existing",
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
       "parameters": {
           "name": "my-cluster"
       }
   }`)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}


func TestUpdate(t *testing.T) {
	suite := NewUpdateSuite(t)
	defer suite.TearDown()

	resp := suite.CallAPI("PATCH", "oauth/cf-eu10/v2/service_instances/my-instance",
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
       "parameters": {
           "name": "my-cluster"
       }
   }`)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}