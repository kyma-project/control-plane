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
	properties       = "properties"
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

		b.updateControlsOrder(&p.Schemas.Instance)
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

func (b *ServicesEndpoint) updateControlsOrder(schema *domain.ServiceInstanceSchema) error {

	casted, ok := schema.Create.Parameters[ControlsOrderKey].([]interface{})
	if !ok {
		return errors.New("Invalid type of Create _controlsOrder param")
	}

	targetControls, err := appendNonExisting(make(map[string]int), casted)
	if err != nil {
		return errors.Wrap(err, "Error while creating _controlsOrder")
	}

	casted, ok = schema.Update.Parameters[ControlsOrderKey].([]interface{})
	if !ok {
		return errors.New("Invalid type of Update _controlsOrder param")
	}

	targetControls, err = appendNonExisting(targetControls, casted)
	if err != nil {
		return errors.Wrap(err, "Error while creating _controlsOrder")
	}

	inverted := invert(targetControls)

	createProps := schema.Create.Parameters[properties].(map[string]interface{})
	schema.Create.Parameters[ControlsOrderKey], err =
		filterAndOrder(inverted, createProps)
	if err != nil {
		return errors.New("Error while updating Create controlOrder")
	}

	updateProps := schema.Update.Parameters[properties].(map[string]interface{})
	schema.Update.Parameters[ControlsOrderKey], err =
		filterAndOrder(inverted, updateProps)
	if err != nil {
		return errors.New("Error while updating Update controlOrder")
	}

	return nil
}

func appendNonExisting(to map[string]int, from []interface{}) (map[string]int, error) {
	size := len(to)
	for i, v := range from {
		key, ok := v.(string)

		if !ok {
			return nil, errors.Errorf("Invalid value type")
		}

		if _, ok = to[key]; !ok {
			to[key] = size + i
		}
	}

	return to, nil
}

func invert(targetControls map[string]int) []string {
	inverted := make([]string, len(targetControls))

	for key, value := range targetControls {
		inverted[value] = key
	}

	return inverted
}

func filterAndOrder(items []string, included map[string]interface{}) ([]interface{}, error) {
	output := make([]interface{}, 0)
	for i := 0; i < len(items); i++ {
		value := items[i]

		if _, ok := included[value]; ok {
			output = append(output, value)
		}
	}

	return output, nil
}
