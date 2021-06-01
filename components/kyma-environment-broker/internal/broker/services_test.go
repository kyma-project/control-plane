package broker_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
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

	cfg := broker.Config{
		EnablePlans: []string{"gcp", "azure", "openstack", "aws", "free", "azure_ha", "aws_ha"},
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
	assert.Len(t, services[0].Plans, 7)

	assert.Equal(t, name, services[0].Metadata.DisplayName)
	assert.Equal(t, supportURL, services[0].Metadata.SupportUrl)
}
