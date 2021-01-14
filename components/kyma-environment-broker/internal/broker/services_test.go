package broker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServices_Services(t *testing.T) {
	// given
	var (
		name       = "testServiceName"
		supportURL = "example.com/support"
	)
	optComponentsProviderMock := &automock.OptionalComponentNamesProvider{}
	defer optComponentsProviderMock.AssertExpectations(t)

	optComponentsNames := []string{"kiali", "tracing"}
	optComponentsProviderMock.On("GetAllOptionalComponentsNames").Return(optComponentsNames)

	cfg := broker.Config{EnablePlans: []string{"gcp", "azure"}}
	cfg.DisplayName = name
	cfg.SupportUrl = supportURL
	servicesEndpoint := broker.NewServices(cfg, optComponentsProviderMock, logrus.StandardLogger())

	// when
	services, err := servicesEndpoint.Services(context.TODO())

	// then
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Len(t, services[0].Plans, 2)

	assert.Equal(t, name, services[0].Metadata.DisplayName)
	assert.Equal(t, supportURL, services[0].Metadata.SupportUrl)

	// assert provisioning schema
	componentItem := services[0].Plans[0].Schemas.Instance.Create.Parameters["properties"].(map[string]interface{})["components"]
	componentJSON, err := json.Marshal(componentItem)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`
		{
		  "type": "array",
		  "items": {
			  "type": "string",
			  "enum": %s
		  }
		}`, toJSONList(optComponentsNames)), string(componentJSON))
}

func toJSONList(in []string) string {
	return fmt.Sprintf(`["%s"]`, strings.Join(in, `", "`))
}
