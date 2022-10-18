package avs

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

const (
	DefinitionType   = "BASIC"
	interval         = 60
	timeout          = 30000
	contentCheck     = "error"
	contentCheckType = "NOT_CONTAINS"
	threshold        = "30000"
	visibility       = "PUBLIC"
)

const (
	StatusActive      = "ACTIVE"
	StatusMaintenance = "MAINTENANCE"
	StatusInactive    = "INACTIVE"
	StatusRetired     = "RETIRED"
	StatusDeleted     = "DELETED"
)

func ValidStatus(status string) bool {
	switch status {
	case StatusActive, StatusMaintenance, StatusInactive, StatusRetired, StatusDeleted:
		return true
	}

	return false
}

type Tag struct {
	Content      string `json:"content"`
	TagClassId   int    `json:"tag_class_id"`
	TagClassName string `json:"tag_class_name"`
}

type BasicEvaluationCreateRequest struct {
	DefinitionType   string `json:"definition_type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Service          string `json:"service"`
	URL              string `json:"url"`
	CheckType        string `json:"check_type"`
	Interval         int32  `json:"interval"`
	TesterAccessId   int64  `json:"tester_access_id"`
	Timeout          int    `json:"timeout"`
	ReadOnly         bool   `json:"read_only"`
	ContentCheck     string `json:"content_check"`
	ContentCheckType string `json:"content_check_type"`
	Threshold        string `json:"threshold"`
	GroupId          int64  `json:"group_id"`
	Visibility       string `json:"visibility"`
	ParentId         int64  `json:"parent_id"`
	Tags             []*Tag `json:"tags"`
}

type BasicEvaluationCreateResponse struct {
	DefinitionType   string `json:"definition_type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Service          string `json:"service"`
	URL              string `json:"url"`
	CheckType        string `json:"check_type"`
	Interval         int32  `json:"interval"`
	TesterAccessId   int64  `json:"tester_access_id"`
	Timeout          int    `json:"timeout"`
	ReadOnly         bool   `json:"read_only"`
	ContentCheck     string `json:"content_check"`
	ContentCheckType string `json:"content_check_type"`
	Threshold        int64  `json:"threshold"`
	GroupId          int64  `json:"group_id"`
	Visibility       string `json:"visibility"`

	DateCreated                int64  `json:"dateCreated"`
	DateChanged                int64  `json:"dateChanged"`
	Owner                      string `json:"owner"`
	Status                     string `json:"status"`
	Alerts                     []int  `json:"alerts"`
	Tags                       []*Tag `json:"tags"`
	Id                         int64  `json:"id"`
	LegacyCheckId              int64  `json:"legacy_check_id"`
	InternalInterval           int64  `json:"internal_interval"`
	AuthType                   string `json:"auth_type"`
	IndividualOutageEventsOnly bool   `json:"individual_outage_events_only"`
	IdOnTester                 string `json:"id_on_tester"`
}

func newBasicEvaluationCreateRequest(operation internal.Operation, evalTypeSpecificConfig ModelConfigurator, url string) (*BasicEvaluationCreateRequest, error) {

	beName, beDescription := generateNameAndDescription(operation, evalTypeSpecificConfig.ProvideSuffix())

	return &BasicEvaluationCreateRequest{
		DefinitionType:   DefinitionType,
		Name:             beName,
		Description:      beDescription,
		Service:          evalTypeSpecificConfig.ProvideNewOrDefaultServiceName(beName),
		URL:              url,
		CheckType:        evalTypeSpecificConfig.ProvideCheckType(),
		Interval:         interval,
		TesterAccessId:   evalTypeSpecificConfig.ProvideTesterAccessId(operation.ProvisioningParameters),
		Tags:             evalTypeSpecificConfig.ProvideTags(),
		Timeout:          timeout,
		ReadOnly:         false,
		ContentCheck:     contentCheck,
		ContentCheckType: contentCheckType,
		Threshold:        threshold,
		GroupId:          evalTypeSpecificConfig.ProvideGroupId(operation.ProvisioningParameters),
		Visibility:       visibility,
		ParentId:         evalTypeSpecificConfig.ProvideParentId(operation.ProvisioningParameters),
	}, nil
}

func generateNameAndDescription(operation internal.Operation, beType string) (string, string) {
	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	subAccountID := operation.ProvisioningParameters.ErsContext.SubAccountID
	instanceID := operation.InstanceID
	name := operation.ProvisioningParameters.Parameters.Name
	shootName := operation.InstanceDetails.ShootName
	beName := fmt.Sprintf("K8S-%s-Kyma-%s-%s-%s", providerCodeByPlan(operation.ProvisioningParameters.PlanID), beType, instanceID, name)
	beDescription := fmt.Sprintf("{\"instanceName\": \"%s\", \"globalAccountID\": \"%s\", \"subAccountID\": \"%s\", \"instanceID\": \"%s\", \"shootName\": \"%s\"}",
		name, globalAccountID, subAccountID, instanceID, shootName)

	return truncateString(beName, 80), truncateString(beDescription, 255)
}

func providerCodeByPlan(planID string) string {
	switch planID {
	case broker.AWSPlanID:
		return "AWS"
	case broker.GCPPlanID:
		return "GCP"
	case broker.AzurePlanID, broker.AzureLitePlanID:
		return "AZR"
	case broker.TrialPlanID, broker.FreemiumPlanID:
		return "AZR"
	case broker.OpenStackPlanID:
		return "CC"
	default:
		return "AZR"
	}
}

func truncateString(input string, num int) string {
	output := input
	if len(input) > num {
		output = input[0:num]
	}
	return output
}
