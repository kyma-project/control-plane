package ers

// Instance defines a struct for data stored by the ERS
type Instance struct {
	Id              string `json:"id"`
	BrokerId        string `json:"brokerId"`
	GlobalAccountId string `json:"globalAccountGUID"`
	PlanId          string `json:"planId"`
	Migrated        bool   `json:"migrated"`
	Status          string `json:"status"`

	// TODO: define all fields
}
