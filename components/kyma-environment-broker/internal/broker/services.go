package broker

import (
	"context"
	"encoding/json"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

// OptionalComponentNamesProvider provides optional components names
type OptionalComponentNamesProvider interface {
	GetAllOptionalComponentsNames() []string
}

type ServicesEndpoint struct {
	log logrus.FieldLogger
	cfg Service

	enabledPlanIDs map[string]struct{}
}

func NewServices(cfg Config, log logrus.FieldLogger) *ServicesEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range cfg.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ServicesEndpoint{
		log:            log.WithField("service", "ServicesEndpoint"),
		cfg:            cfg.Service,
		enabledPlanIDs: enabledPlanIDs,
	}
}

// Services gets the catalog of services offered by the service broker
//   GET /v2/catalog
func (b *ServicesEndpoint) Services(ctx context.Context) ([]domain.Service, error) {
	var availableServicePlans []domain.ServicePlan

	for _, plan := range Plans {
		// filter out not enabled plans
		if _, exists := b.enabledPlanIDs[plan.PlanDefinition.ID]; !exists {
			continue
		}
		p := plan.PlanDefinition
		err := json.Unmarshal(plan.provisioningRawSchema, &p.Schemas.Instance.Create.Parameters)
		if err != nil {
			b.log.Errorf("while unmarshal schema: %s", err)
			return nil, err
		}
		availableServicePlans = append(availableServicePlans, p)
	}

	return []domain.Service{
		{
			ID:                   KymaServiceID,
			Name:                 KymaServiceName,
			Description:          "[EXPERIMENTAL] Service Class for Kyma Runtime",
			Bindable:             true,
			InstancesRetrievable: true,
			Tags: []string{
				"SAP",
				"Kyma",
			},
			Plans: availableServicePlans,
			Metadata: &domain.ServiceMetadata{
				DisplayName:         b.cfg.DisplayName,
				ImageUrl:            b.cfg.ImageUrl,
				LongDescription:     b.cfg.LongDescription,
				ProviderDisplayName: b.cfg.ProviderDisplayName,
				DocumentationUrl:    b.cfg.DocumentationUrl,
				SupportUrl:          b.cfg.SupportUrl,
			},
			AllowContextUpdates: true,
		},
	}, nil
}
