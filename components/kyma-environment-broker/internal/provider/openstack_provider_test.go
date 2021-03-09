package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
)

func TestZonesForOpenStackZones(t *testing.T) {
	regions := broker.OpenStackRegions()
	for _, region := range regions {
		_, exists := openstackZones[region]
		assert.True(t, exists)
	}
	_, exists := openstackZones[DefaultOpenStackRegion]
	assert.True(t, exists)
}
