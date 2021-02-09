package broker

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	KymaServiceName = "kymaruntime"
)

type KymaEnvironmentBroker struct {
	*ServicesEndpoint
	*ProvisionEndpoint
	*DeprovisionEndpoint
	*UpdateEndpoint
	*GetInstanceEndpoint
	*LastOperationEndpoint
	*BindEndpoint
	*UnbindEndpoint
	*GetBindingEndpoint
	*LastBindingOperationEndpoint
}

// Config represents configuration for broker
type Config struct {
	Service
	EnablePlans          EnablePlans `envconfig:"default=azure"`
	OnlySingleTrialPerGA bool        `envconfig:"default=true"`
}

type Service struct {
	DisplayName         string
	ImageUrl            string
	LongDescription     string
	ProviderDisplayName string
	DocumentationUrl    string
	SupportUrl          string
}

// EnablePlans defines the plans that should be available for provisioning
type EnablePlans []string

// Unmarshal provides custom parsing of enabled plans.
// Implements envconfig.Unmarshal interface.
func (m *EnablePlans) Unmarshal(in string) error {
	plans := strings.Split(in, ",")
	for _, name := range plans {
		if _, exists := PlanIDsMapping[name]; !exists {
			return errors.Errorf("unrecognized %v plan name ", name)
		}
	}

	*m = plans
	return nil
}
