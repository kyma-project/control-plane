package internal

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type ProvisionerInputCreator interface {
	SetProvisioningParameters(params ProvisioningParameters) ProvisionerInputCreator
	SetShootName(string) ProvisionerInputCreator
	SetLabel(key, value string) ProvisionerInputCreator
	// Deprecated, use: AppendOverrides
	SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator
	AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator
	AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator
	CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error)
	CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error)
	EnableOptionalComponent(componentName string) ProvisionerInputCreator
}

type LMSTenant struct {
	ID        string
	Name      string
	Region    string
	CreatedAt time.Time
}

type LMS struct {
	TenantID    string    `json:"tenant_id"`
	Failed      bool      `json:"failed"`
	RequestedAt time.Time `json:"requested_at"`
}

type AvsLifecycleData struct {
	AvsEvaluationInternalId int64 `json:"avs_evaluation_internal_id"`
	AVSEvaluationExternalId int64 `json:"avs_evaluation_external_id"`

	AvsInternalEvaluationStatus avs.Status `json:"avs_internal_evaluation_status"`
	AvsExternalEvaluationStatus avs.Status `json:"avs_external_evaluation_status"`

	AvsOriginalInternalEvaluationStatus avs.Status `json:"avs_original_internal_evaluation_status"`
	AvsOriginalExternalEvaluationStatus avs.Status `json:"avs_original_external_evaluation_status"`

	AVSInternalEvaluationDeleted bool `json:"avs_internal_evaluation_deleted"`
	AVSExternalEvaluationDeleted bool `json:"avs_external_evaluation_deleted"`
}

// RuntimeVersionOrigin defines the possible sources of the Kyma Version parameter
type RuntimeVersionOrigin string

const (
	Parameters     RuntimeVersionOrigin = "parameters"
	Defaults       RuntimeVersionOrigin = "defaults"
	AccountMapping RuntimeVersionOrigin = "account-mapping"
)

// RuntimeVersionData describes the Kyma Version used for the cluser
// provisioning or upgrade
type RuntimeVersionData struct {
	Version string               `json:"version"`
	Origin  RuntimeVersionOrigin `json:"origin"`
}

func (rv RuntimeVersionData) IsEmpty() bool {
	return rv.Version == ""
}

func NewRuntimeVersionFromParameters(version string) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: Parameters}
}

func NewRuntimeVersionFromDefaults(version string) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: Defaults}
}

func NewRuntimeVersionFromAccountMapping(version string) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: AccountMapping}
}

type EventHub struct {
	Deleted bool `json:"event_hub_deleted"`
}

type Instance struct {
	InstanceID      string
	RuntimeID       string
	GlobalAccountID string
	SubAccountID    string
	ServiceID       string
	ServiceName     string
	ServicePlanID   string
	ServicePlanName string

	DashboardURL           string
	ProvisioningParameters string
	ProviderRegion         string

	InstanceDetails InstanceDetails

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time

	Version int
}

func (instance Instance) GetProvisioningParameters() (ProvisioningParameters, error) {
	var pp ProvisioningParameters

	err := json.Unmarshal([]byte(instance.ProvisioningParameters), &pp)
	if err != nil {
		return pp, errors.Wrap(err, "while unmarshalling provisioning parameters")
	}

	return pp, nil
}

type Operation struct {
	// following fields are serialized to JSON and stored in the storage
	InstanceDetails

	ID        string    `json:"-"`
	Version   int       `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	InstanceID             string                    `json:"-"`
	ProvisionerOperationID string                    `json:"-"`
	State                  domain.LastOperationState `json:"-"`
	Description            string                    `json:"-"`
	ProvisioningParameters ProvisioningParameters    `json:"-"`

	// OrchestrationID specifies the origin orchestration which triggers the operation, empty for OSB operations (provisioning/deprovisioning)
	OrchestrationID string `json:"-"`
}

func (o *Operation) IsFinished() bool {
	return o.State != orchestration.InProgress && o.State != orchestration.Pending && o.State != orchestration.Canceled
}

// todo: remove after parameters migration was done on each environment
// LegacyOperation represents old structure of the Operation struct which now has provisioning parameters inside
type LegacyOperation struct {
	Operation `json:"-"`

	Type                   string `json:"type"`
	ProvisioningParameters string `json:"provisioning_parameters"`
}

// Orchestration holds all information about an orchestration.
// Orchestration performs operations of a specific type (UpgradeKymaOperation, UpgradeClusterOperation)
// on specific targets of SKRs.
type Orchestration struct {
	OrchestrationID string
	State           string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Parameters      orchestration.Parameters
}

func (o *Orchestration) IsFinished() bool {
	return o.State == orchestration.Succeeded || o.State == orchestration.Failed || o.State == orchestration.Canceled
}

type InstanceWithOperation struct {
	Instance

	Type        sql.NullString
	State       sql.NullString
	Description sql.NullString
}

type SMClientFactory interface {
	ForCustomerCredentials(reqCredentials *servicemanager.Credentials, log logrus.FieldLogger) (servicemanager.Client, error)
	ProvideCredentials(reqCredentials *servicemanager.Credentials, log logrus.FieldLogger) (*servicemanager.Credentials, error)
}

type InstanceDetails struct {
	Lms LMS `json:"lms"`

	Avs      AvsLifecycleData `json:"avs"`
	EventHub EventHub         `json:"eh"`

	SubAccountID string    `json:"sub_account_id"`
	RuntimeID    string    `json:"runtime_id"`
	ShootName    string    `json:"shoot_name"`
	ShootDomain  string    `json:"shoot_domain"`
	XSUAA        XSUAAData `json:"xsuaa"`
	Ems          EmsData   `json:"ems"`
}

// ProvisioningOperation holds all information about provisioning operation
type ProvisioningOperation struct {
	Operation

	RuntimeVersion RuntimeVersionData `json:"runtime_version"`

	// following fields are not stored in the storage
	InputCreator ProvisionerInputCreator `json:"-"`

	SMClientFactory SMClientFactory `json:"-"`
}

type ServiceManagerInstanceInfo struct {
	BrokerID              string `json:"brokerId"`
	ServiceID             string `json:"serviceId"`
	PlanID                string `json:"planId"` // it is a plan.CatalogID from the Service Manager perspective
	InstanceID            string `json:"instanceId"`
	Provisioned           bool   `json:"provisioned"`
	ProvisioningTriggered bool   `json:"provisioningTriggered"`
}

type XSUAAData struct {
	Instance ServiceManagerInstanceInfo `json:"instance"`

	XSAppname string `json:"xsappname"`
	BindingID string `json:"bindingId"`
}

type EmsData struct {
	Instance ServiceManagerInstanceInfo `json:"instance"`

	BindingID string `json:"bindingId"`
	Overrides string `json:"overrides"`
}

func (s *ServiceManagerInstanceInfo) InstanceKey() servicemanager.InstanceKey {
	return servicemanager.InstanceKey{
		BrokerID:   s.BrokerID,
		InstanceID: s.InstanceID,
		ServiceID:  s.ServiceID,
		PlanID:     s.PlanID,
	}
}

// DeprovisioningOperation holds all information about de-provisioning operation
type DeprovisioningOperation struct {
	Operation

	SMClientFactory SMClientFactory `json:"-"`
}

// UpgradeKymaOperation holds all information about upgrade Kyma operation
type UpgradeKymaOperation struct {
	Operation

	orchestration.RuntimeOperation `json:"runtime_operation"`
	InputCreator                   ProvisionerInputCreator `json:"-"`

	RuntimeVersion RuntimeVersionData `json:"runtime_version"`
}

func NewRuntimeState(runtimeID, operationID string, kymaConfig *gqlschema.KymaConfigInput, clusterConfig *gqlschema.GardenerConfigInput) RuntimeState {
	var (
		kymaConfigInput    gqlschema.KymaConfigInput
		clusterConfigInput gqlschema.GardenerConfigInput
	)
	if kymaConfig != nil {
		kymaConfigInput = *kymaConfig
	}
	if clusterConfig != nil {
		clusterConfigInput = *clusterConfig
	}

	return RuntimeState{
		ID:            uuid.New().String(),
		CreatedAt:     time.Now(),
		RuntimeID:     runtimeID,
		OperationID:   operationID,
		KymaConfig:    kymaConfigInput,
		ClusterConfig: clusterConfigInput,
	}
}

type RuntimeState struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`

	KymaConfig    gqlschema.KymaConfigInput     `json:"kymaConfig"`
	ClusterConfig gqlschema.GardenerConfigInput `json:"clusterConfig"`
}

// OperationStats provide number of operations per type and state
type OperationStats struct {
	Provisioning   map[domain.LastOperationState]int
	Deprovisioning map[domain.LastOperationState]int
}

// InstanceStats provide number of instances per Global Account ID
type InstanceStats struct {
	TotalNumberOfInstances int
	PerGlobalAccountID     map[string]int
}

// NewProvisioningOperation creates a fresh (just starting) instance of the ProvisioningOperation
func NewProvisioningOperation(instanceID string, parameters ProvisioningParameters) (ProvisioningOperation, error) {
	return NewProvisioningOperationWithID(uuid.New().String(), instanceID, parameters)
}

// NewProvisioningOperationWithID creates a fresh (just starting) instance of the ProvisioningOperation with provided ID
func NewProvisioningOperationWithID(operationID, instanceID string, parameters ProvisioningParameters) (ProvisioningOperation, error) {
	return ProvisioningOperation{
		Operation: Operation{
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instanceID,
			State:                  domain.InProgress,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			ProvisioningParameters: parameters,
			InstanceDetails: InstanceDetails{
				SubAccountID: parameters.ErsContext.SubAccountID,
			},
		},
	}, nil
}

// NewProvisioningOperationWithID creates a fresh (just starting) instance of the ProvisioningOperation with provided ID
func NewDeprovisioningOperationWithID(operationID, instanceID string) (DeprovisioningOperation, error) {
	return DeprovisioningOperation{
		Operation: Operation{
			ID:          operationID,
			Version:     0,
			Description: "Operation created",
			InstanceID:  instanceID,
			State:       domain.InProgress,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}, nil
}

func (po *ProvisioningOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	return po.SMClientFactory.ForCustomerCredentials(serviceManagerRequestCreds(po.ProvisioningParameters), log)
}

func (po *ProvisioningOperation) ProvideServiceManagerCredentials(log logrus.FieldLogger) (*servicemanager.Credentials, error) {
	return po.SMClientFactory.ProvideCredentials(serviceManagerRequestCreds(po.ProvisioningParameters), log)
}

func (do *DeprovisioningOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	return do.SMClientFactory.ForCustomerCredentials(serviceManagerRequestCreds(do.ProvisioningParameters), log)
}

type ComponentConfigurationInputList []*gqlschema.ComponentConfigurationInput

func (l ComponentConfigurationInputList) DeepCopy() []*gqlschema.ComponentConfigurationInput {
	var copiedList []*gqlschema.ComponentConfigurationInput
	for _, component := range l {
		var cpyCfg []*gqlschema.ConfigEntryInput
		for _, cfg := range component.Configuration {
			mapped := &gqlschema.ConfigEntryInput{
				Key:   cfg.Key,
				Value: cfg.Value,
			}
			if cfg.Secret != nil {
				mapped.Secret = ptr.Bool(*cfg.Secret)
			}
			cpyCfg = append(cpyCfg, mapped)
		}

		copiedList = append(copiedList, &gqlschema.ComponentConfigurationInput{
			Component:     component.Component,
			Namespace:     component.Namespace,
			SourceURL:     component.SourceURL,
			Configuration: cpyCfg,
		})
	}
	return copiedList
}

func serviceManagerRequestCreds(parameters ProvisioningParameters) *servicemanager.Credentials {
	var creds *servicemanager.Credentials
	sm := parameters.ErsContext.ServiceManager
	if sm != nil {
		creds = &servicemanager.Credentials{
			Username: sm.Credentials.BasicAuth.Username,
			Password: sm.Credentials.BasicAuth.Password,
			URL:      sm.URL,
		}
	}
	return creds
}
