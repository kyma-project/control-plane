package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
)

func TestAWSZones(t *testing.T) {
	regions := broker.AWSRegions()
	for _, region := range regions {
		_, exists := awsZones[region]
		assert.True(t, exists)
	}
	_, exists := awsZones[DefaultAWSRegion]
	assert.True(t, exists)
}
