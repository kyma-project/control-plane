package broker

import (
	"context"
	"encoding/json"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/middleware"

	"github.com/pkg/errors"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

const (
	ControlsOrderKey = "_controlsOrder"
	PropertiesKey    = "properties"
)

type ServicesEndpoint struct {
	log            logrus.FieldLogger
	cfg            Config
	servicesConfig ServicesConfig

	enabledPlanIDs map[string]struct{}
}

func NewServices(cfg Config, servicesConfig ServicesConfig, log logrus.FieldLogger) *ServicesEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range cfg.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ServicesEndpoint{
		log:            log.WithField("service", "ServicesEndpoint"),
		cfg:            cfg,
		servicesConfig: servicesConfig,
		enabledPlanIDs: enabledPlanIDs,
	}
}

// Services gets the catalog of services offered by the service broker
//   GET /v2/catalog
func (b *ServicesEndpoint) Services(ctx context.Context) ([]domain.Service, error) {
	var availableServicePlans []domain.ServicePlan
	// we scope to the kymaruntime service only
	class, ok := b.servicesConfig[KymaServiceName]
	if !ok {
		return nil, errors.Errorf("while getting %s class data", KymaServiceName)
	}

	provider, ok := middleware.ProviderFromContext(ctx)
	for _, plan := range Plans(class.Plans, provider, b.cfg.IncludeAdditionalParamsInSchema) {
		// filter out not enabled plans
		if _, exists := b.enabledPlanIDs[plan.PlanDefinition.ID]; !exists {
			continue
		}
		p := plan.PlanDefinition

		err := json.Unmarshal(plan.provisioningRawSchema, &p.Schemas.Instance.Create.Parameters)
		if err != nil {
			b.log.Errorf("while unmarshal provisioning schema: %s", err)
			return nil, err
		}

		if len(plan.catalogRawSchema) > 0 {
			// overwrite provisioning parameters schema if Plan.catalogRawSchema is provided
			err := json.Unmarshal(plan.catalogRawSchema, &p.Schemas.Instance.Create.Parameters)
			if err != nil {
				b.log.Errorf("while unmarshal provisioning schema: %s", err)
				return nil, err
			}
		}

		if len(plan.updateRawSchema) > 0 {
			err = json.Unmarshal(plan.updateRawSchema, &p.Schemas.Instance.Update.Parameters)
			if err != nil {
				b.log.Errorf("while unmarshal update schema: %s", err)
				return nil, err
			}
		}

		availableServicePlans = append(availableServicePlans, p)
	}

	return []domain.Service{
		{
			ID:                   KymaServiceID,
			Name:                 KymaServiceName,
			Description:          class.Description,
			Bindable:             false,
			InstancesRetrievable: true,
			Tags: []string{
				"SAP",
				"Kyma",
			},
			Plans: availableServicePlans,
			Metadata: &domain.ServiceMetadata{
				DisplayName:         class.Metadata.DisplayName,
				ImageUrl:            class.Metadata.ImageUrl,
				LongDescription:     class.Metadata.LongDescription,
				ProviderDisplayName: class.Metadata.ProviderDisplayName,
				DocumentationUrl:    class.Metadata.DocumentationUrl,
				SupportUrl:          class.Metadata.SupportUrl,
			},
			AllowContextUpdates: true,
		},
	}, nil
}
