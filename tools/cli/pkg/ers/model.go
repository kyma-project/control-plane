package ers

import (
	"fmt"
	"time"
)

type Work struct {
	Instance           Instance
	MigrationMetadata  MigrationMetadata
	ProcessedTimestamp int64
	ProcessedCnt       int64
	MaxProcessedCnt    int64
}
type Instance struct {
	BrokerId       string
	CommercialType string
	// TODO: Date comes in different format than specified
	// CreatedDate       date.Date // 2021-10-04T112314.137Z
	CustomLabels    interface{}
	DashboardUrl    string
	Description     string
	EnvironmentType string // always "kyma"
	GlobalAccountID string `json:"globalAccountGUID"`
	Id              string
	Labels          string
	LandscapeLabel  string
	// TODO: Date comes in different format than specified
	// ModifiedDate      date.Date //2021-10-04T112314.137Z
	Name           string
	Operation      string
	Parameters     interface{}
	PlanId         string
	PlanName       string
	PlatformId     string
	ServiceId      string
	ServiceName    string
	State          string //[CREATING UPDATING DELETING OK CREATION_FAILED DELETION_FAILED UPDATE_FAILED]
	StateMessage   string
	SubaccountGUID string
	TenantId       string
	Type           string // always "Provision"
	Status         string // always "Processed"
	Migrated       bool
}

func (i *Instance) IsUsable() bool {
	//return i.State != "CREATION_FAILED" && i.State != "DELETION_FAILED"
	return true
}

type MigrationMetadata struct {
	Id                      string    `json:"id"`
	KymaMigrated            bool      `json:"kymaMigrated"`
	KymaSkipped             bool      `json:"kymaSkipped"`
	KymaMigrationStartedAt  time.Time `json:"kymaMigrationStartedAt"`
	KymaMigrationFinishedAt time.Time `json:"kymaMigrationFinishedAt"`
}

func (e Instance) String() string {
	return fmt.Sprintf(`BrokerId: %s
	CommercialType %s
	CustomLabels %s
	DashboardUrl %s
	Description %s
	EnvironmentType %s
	GlobalAccountId %s
	Id %s
	Labels %s
	LandscapeLabel %s
	Name %s
	Operation %s
	Parameters %s
	PlanId %s
	PlanName %s
	PlatformId %s
	ServiceId %s
	ServiceName %s
	State %s //[CREATING UPDATING DELETING OK CREATION_FAILED DELETION_FAILED UPDATE_FAILED]
	StateMessage %s
	SubaccountGUID %s
	TenantId %s
	Type %s
	Status %s // always Processed
	Migrated %v`,
		e.BrokerId, e.CommercialType, e.CustomLabels, e.DashboardUrl, e.Description, e.EnvironmentType, e.GlobalAccountID,
		e.Id, e.Labels, e.LandscapeLabel, e.Name, e.Operation, e.Parameters, e.PlanId, e.PlanName, e.PlatformId, e.ServiceId, e.ServiceName,
		e.State, e.StateMessage, e.SubaccountGUID, e.TenantId, e.Type, e.Status, e.Migrated)
}
