package keb

import "time"

type Provisioning struct {
	State       string
	Description string
	CreatedAt   time.Time
	OperationID string
}

type Deprovisioning struct {
	State       string
	Description string
	CreatedAt   time.Time
	OperationID string
}

type UpgradingKyma struct {
	Data       []string
	TotalCount int
	Count      int
}

type Status struct {
	CreatedAt      time.Time
	ModifiedAt     time.Time
	Provisioning   Provisioning
	Deprovisioning Deprovisioning
	UpgradingKyma  UpgradingKyma
}

type Runtime struct {
	InstanceID       string
	RuntimeID        string
	GlobalAccountID  string
	SubAccountID     string
	Region           string
	SubAccountRegion string
	ShootName        string
	ServiceClassID   string
	ServiceClassName string
	ServicePlanID    string
	ServicePlanName  string
	Status           Status
}

type Runtimes struct {
	Data       []Runtime
	Count      int
	TotalCount int
}