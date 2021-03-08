package servicemanager

import "fmt"

type OfferingInfo struct {
	ServiceID string
	BrokerID  string
	PlanID    string
}

func GenerateOfferingInfo(c Client, offeringName, planName string) (*OfferingInfo, bool, error) {
	var info OfferingInfo
	// try to find the offering
	offerings, err := c.ListOfferingsByName(offeringName)
	if err != nil {
		return &info, true, fmt.Errorf("unable to get Service Manager offerings: %v", err)
	}
	if len(offerings.ServiceOfferings) != 1 {
		return &info, false,
			fmt.Errorf("expected one %s Service Manager offering, but found %d", offeringName, len(offerings.ServiceOfferings))
	}
	info.ServiceID = offerings.ServiceOfferings[0].CatalogID
	info.BrokerID = offerings.ServiceOfferings[0].BrokerID

	// try to find the plan
	plans, err := c.ListPlansByName(planName, offerings.ServiceOfferings[0].ID)
	if err != nil {
		return &info, true, fmt.Errorf("unable to get Service Manager plan: %v", err)
	}
	if len(plans.ServicePlans) != 1 {
		return &info, false,
			fmt.Errorf("expected one %s Service Manager plan, but found %d", planName, len(offerings.ServiceOfferings))
	}

	info.PlanID = plans.ServicePlans[0].CatalogID

	return &info, false, nil
}
