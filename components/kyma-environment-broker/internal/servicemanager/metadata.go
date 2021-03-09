package servicemanager

import (
	"fmt"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

type Metadata struct {
	ServiceID string
	BrokerID  string
	PlanID    string
}

func GenerateMetadata(c Client, offeringName, planName string) (*Metadata, error) {
	var meta Metadata
	// try to find the offering
	offerings, err := c.ListOfferingsByName(offeringName)
	if err != nil {
		return &meta, kebError.AsTemporaryError(err, "while getting Service Manager offerings")
	}
	if len(offerings.ServiceOfferings) != 1 {
		return &meta,
			fmt.Errorf("expected one %s Service Manager offering, but found %d", offeringName, len(offerings.ServiceOfferings))
	}
	meta.ServiceID = offerings.ServiceOfferings[0].CatalogID
	meta.BrokerID = offerings.ServiceOfferings[0].BrokerID

	// try to find the plan
	plans, err := c.ListPlansByName(planName, offerings.ServiceOfferings[0].ID)
	if err != nil {
		return &meta, kebError.AsTemporaryError(err, "while getting Service Manager plan")
	}
	if len(plans.ServicePlans) != 1 {
		return &meta,
			fmt.Errorf("expected one %s Service Manager plan, but found %d", planName, len(offerings.ServiceOfferings))
	}

	meta.PlanID = plans.ServicePlans[0].CatalogID

	return &meta, nil
}
