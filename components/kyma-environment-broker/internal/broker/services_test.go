package broker_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServices_Services(t *testing.T) {

	t.Run("should get service and plans without OIDC", func(t *testing.T) {
		// given
		var (
			name       = "testServiceName"
			supportURL = "example.com/support"
		)

		cfg := broker.Config{
			EnablePlans: []string{"gcp", "azure", "openstack", "aws", "free"},
		}
		servicesConfig := map[string]broker.Service{
			broker.KymaServiceName: {
				Metadata: broker.ServiceMetadata{
					DisplayName: name,
					SupportUrl:  supportURL,
				},
			},
		}
		servicesEndpoint := broker.NewServices(cfg, servicesConfig, logrus.StandardLogger())

		// when
		services, err := servicesEndpoint.Services(context.TODO())

		// then
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Len(t, services[0].Plans, 5)

		assert.Equal(t, name, services[0].Metadata.DisplayName)
		assert.Equal(t, supportURL, services[0].Metadata.SupportUrl)
	})
	t.Run("should get service and plans with OIDC & administrators", func(t *testing.T) {
		// given
		var (
			name       = "testServiceName"
			supportURL = "example.com/support"
		)

		cfg := broker.Config{
			EnablePlans:                     []string{"gcp", "azure", "openstack", "aws", "free"},
			IncludeAdditionalParamsInSchema: true,
		}
		servicesConfig := map[string]broker.Service{
			broker.KymaServiceName: {
				Metadata: broker.ServiceMetadata{
					DisplayName: name,
					SupportUrl:  supportURL,
				},
			},
		}
		servicesEndpoint := broker.NewServices(cfg, servicesConfig, logrus.StandardLogger())

		// when
		services, err := servicesEndpoint.Services(context.TODO())

		// then
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Len(t, services[0].Plans, 5)

		assert.Equal(t, name, services[0].Metadata.DisplayName)
		assert.Equal(t, supportURL, services[0].Metadata.SupportUrl)

		assertPlansContainPropertyInSchemas(t, services[0], "oidc")
		assertPlansContainPropertyInSchemas(t, services[0], "administrators")
	})

	t.Run("should return sync control orders", func(t *testing.T) {
		// given
		var (
			name       = "testServiceName"
			supportURL = "example.com/support"
		)

		cfg := broker.Config{
			EnablePlans:                     []string{"gcp", "azure", "openstack", "aws", "free"},
			IncludeAdditionalParamsInSchema: true,
		}
		servicesConfig := map[string]broker.Service{
			broker.KymaServiceName: {
				Metadata: broker.ServiceMetadata{
					DisplayName: name,
					SupportUrl:  supportURL,
				},
			},
		}
		servicesEndpoint := broker.NewServices(cfg, servicesConfig, logrus.StandardLogger())

		// when
		services, err := servicesEndpoint.Services(context.TODO())

		// then
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Len(t, services[0].Plans, 5)

		assert.Equal(t, name, services[0].Metadata.DisplayName)
		assert.Equal(t, supportURL, services[0].Metadata.SupportUrl)

		assertPlansContainPropertyInSchemas(t, services[0], "oidc")
		assertPlansContainPropertyInSchemas(t, services[0], "administrators")
	})
}

func assertPlansContainPropertyInSchemas(t *testing.T, service domain.Service, property string) {
	for _, plan := range service.Plans {
		assertPlanContainsPropertyInCreateSchema(t, plan, property)
		assertPlanContainsPropertyInUpdateSchema(t, plan, property)
	}
}

func assertPlanContainsPropertyInCreateSchema(t *testing.T, plan domain.ServicePlan, property string) {
	properties := plan.Schemas.Instance.Create.Parameters[broker.PropertiesKey]
	propertiesMap := properties.(map[string]interface{})
	if _, exists := propertiesMap[property]; !exists {
		t.Errorf("plan %s does not contain %s property in Create schema", plan.Name, property)
	}
}

func assertPlanContainsPropertyInUpdateSchema(t *testing.T, plan domain.ServicePlan, property string) {
	properties := plan.Schemas.Instance.Update.Parameters[broker.PropertiesKey]
	propertiesMap := properties.(map[string]interface{})
	if _, exists := propertiesMap[property]; !exists {
		t.Errorf("plan %s does not contain %s property in Update schema", plan.Name, property)
	}
}
