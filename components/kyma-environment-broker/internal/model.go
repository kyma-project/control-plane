package internal

import (
	"database/sql"
	"encoding/json"
	"time"

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

	AVSInternalEvaluationDeleted bool `json:"avs_internal_evaluation_deleted"`
	AVSExternalEvaluationDeleted bool `json:"avs_external_evaluation_deleted"`
}

// RuntimeVersionOrigin defines the possible sources of the Kyma Version parameter
type RuntimeVersionOrigin string

const (
	Parameters    RuntimeVersionOrigin = "parameters"
	Defaults      RuntimeVersionOrigin = "defaults"
	GlobalAccount RuntimeVersionOrigin = "global-account"
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

func NewRuntimeVersionFromGlobalAccount(version string) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: GlobalAccount}
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

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
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
	ID        string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time

	InstanceID             string
	ProvisionerOperationID string
	State                  domain.LastOperationState
	Description            string

	// OrchestrationID specifies the origin orchestration which triggers the operation, empty for OSB operations (provisioning/deprovisioning)
	OrchestrationID string
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

// ProvisioningOperation holds all information about provisioning operation
type ProvisioningOperation struct {
	Operation       `json:"-"`
	SMClientFactory SMClientFactory `json:"-"`

	// following fields are serialized to JSON and stored in the storage
	Lms                    LMS    `json:"lms"`
	ProvisioningParameters string `json:"provisioning_parameters"`

	// following fields are not stored in the storage
	InputCreator ProvisionerInputCreator `json:"-"`

	Avs AvsLifecycleData `json:"avs"`

	RuntimeID string `json:"runtime_id"`

	RuntimeVersion RuntimeVersionData `json:"runtime_version"`
}

// DeprovisioningOperation holds all information about de-provisioning operation
type DeprovisioningOperation struct {
	Operation `json:"-"`

	ProvisioningParameters string           `json:"provisioning_parameters"`
	Avs                    AvsLifecycleData `json:"avs"`
	EventHub               EventHub         `json:"eh"`
	SubAccountID           string           `json:"-"`
	RuntimeID              string           `json:"runtime_id"`
}

// UpgradeKymaOperation holds all information about upgrade Kyma operation
type UpgradeKymaOperation struct {
	Operation                      `json:"-"`
	orchestration.RuntimeOperation `json:"runtime_operation"`
	InputCreator                   ProvisionerInputCreator `json:"-"`

	PlanID                 string `json:"plan_id"`
	ProvisioningParameters string `json:"provisioning_parameters"`

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
	params, err := json.Marshal(parameters)
	if err != nil {
		return ProvisioningOperation{}, errors.Wrap(err, "while marshaling provisioning parameters")
	}

	return ProvisioningOperation{
		Operation: Operation{
			ID:          operationID,
			Version:     0,
			Description: "Operation created",
			InstanceID:  instanceID,
			State:       domain.InProgress,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		ProvisioningParameters: string(params),
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

func (po *ProvisioningOperation) GetProvisioningParameters() (ProvisioningParameters, error) {
	var pp ProvisioningParameters

	err := json.Unmarshal([]byte(po.ProvisioningParameters), &pp)
	if err != nil {
		return pp, errors.Wrapf(err, "while unmarshaling provisioning parameters: %s, ProvisioningOperations: %+v", po.ProvisioningParameters, po)
	}

	return pp, nil
}

func (po *ProvisioningOperation) SetProvisioningParameters(parameters ProvisioningParameters) error {
	params, err := json.Marshal(parameters)
	if err != nil {
		return errors.Wrap(err, "while marshaling provisioning parameters")
	}

	po.ProvisioningParameters = string(params)
	return nil
}

func (po *ProvisioningOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	pp, err := po.GetProvisioningParameters()
	if err != nil {
		log.Errorf("unable to get Provisioning Parameters: %s", err.Error())
		return nil, errors.New("invalid operation provisioning parameters")
	}

	return po.SMClientFactory.ForCustomerCredentials(po.serviceManagerRequestCereds(pp), log)
}

func (po *ProvisioningOperation) ProvideServiceManagerCredentials(log logrus.FieldLogger) (*servicemanager.Credentials, error) {
	pp, err := po.GetProvisioningParameters()
	if err != nil {
		log.Errorf("unable to get Provisioning Parameters: %s", err.Error())
		return nil, errors.New("invalid operation provisioning parameters")
	}

	return po.SMClientFactory.ProvideCredentials(po.serviceManagerRequestCereds(pp), log)
}

func (po *ProvisioningOperation) serviceManagerRequestCereds(parameters ProvisioningParameters) *servicemanager.Credentials {
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

func (do *DeprovisioningOperation) GetProvisioningParameters() (ProvisioningParameters, error) {
	var pp ProvisioningParameters

	err := json.Unmarshal([]byte(do.ProvisioningParameters), &pp)
	if err != nil {
		return pp, errors.Wrapf(err, "while unmarshaling provisioning parameters: %s, ProvisioningOperations: %+v", do.ProvisioningParameters, do)
	}

	return pp, nil
}

func (do *DeprovisioningOperation) SetProvisioningParameters(parameters ProvisioningParameters) error {
	params, err := json.Marshal(parameters)
	if err != nil {
		return errors.Wrap(err, "while marshaling provisioning parameters")
	}

	do.ProvisioningParameters = string(params)
	return nil
}

func (do *UpgradeKymaOperation) GetProvisioningParameters() (ProvisioningParameters, error) {
	var pp ProvisioningParameters

	err := json.Unmarshal([]byte(do.ProvisioningParameters), &pp)
	if err != nil {
		return pp, errors.Wrapf(err, "while unmarshaling provisioning parameters: %s, ProvisioningOperations: %+v", do.ProvisioningParameters, do)
	}

	return pp, nil
}

func (do *UpgradeKymaOperation) SetProvisioningParameters(parameters ProvisioningParameters) error {
	params, err := json.Marshal(parameters)
	if err != nil {
		return errors.Wrap(err, "while marshaling provisioning parameters")
	}

	do.ProvisioningParameters = string(params)
	return nil
}

func (o *Operation) IsFinished() bool {
	return o.State != domain.InProgress && o.State != orchestration.Pending
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
