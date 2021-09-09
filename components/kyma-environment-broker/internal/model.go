package internal

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
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
	CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error)
	EnableOptionalComponent(componentName string) ProvisionerInputCreator
	Provider() CloudProvider

	CreateProvisionSKRInventoryInput() (reconciler.Cluster, error)
	CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error)
	SetKubeconfig(kcfg string) ProvisionerInputCreator
	SetRuntimeID(runtimeID string) ProvisionerInputCreator
	SetInstanceID(instanceID string) ProvisionerInputCreator
}

// GitKymaProject and GitKymaRepo define public Kyma GitHub parameters used for
// external evaluation.
const (
	GitKymaProject = "kyma-project"
	GitKymaRepo    = "kyma"
)

type AvsEvaluationStatus struct {
	Current  string `json:"current_value"`
	Original string `json:"original_value"`
}

type AvsLifecycleData struct {
	AvsEvaluationInternalId int64 `json:"avs_evaluation_internal_id"`
	AVSEvaluationExternalId int64 `json:"avs_evaluation_external_id"`

	AvsInternalEvaluationStatus AvsEvaluationStatus `json:"avs_internal_evaluation_status"`
	AvsExternalEvaluationStatus AvsEvaluationStatus `json:"avs_external_evaluation_status"`

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

// RuntimeVersionData describes the Kyma Version used for the cluster
// provisioning or upgrade
type RuntimeVersionData struct {
	Version      string               `json:"version"`
	Origin       RuntimeVersionOrigin `json:"origin"`
	MajorVersion int                  `json:"major_version"`
}

func (rv RuntimeVersionData) IsEmpty() bool {
	return rv.Version == ""
}

func NewRuntimeVersionFromParameters(version string, majorVersion int) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: Parameters, MajorVersion: majorVersion}
}

func NewRuntimeVersionFromDefaults(version string) *RuntimeVersionData {
	defaultMajorVerNum, _ := strconv.Atoi(string(version[0]))
	return &RuntimeVersionData{Version: version, Origin: Defaults, MajorVersion: defaultMajorVerNum}
}

func NewRuntimeVersionFromAccountMapping(version string, majorVersion int) *RuntimeVersionData {
	return &RuntimeVersionData{Version: version, Origin: AccountMapping, MajorVersion: majorVersion}
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

	DashboardURL   string
	Parameters     ProvisioningParameters
	ProviderRegion string

	InstanceDetails InstanceDetails

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time

	Version  int
	Provider CloudProvider
}

func (i *Instance) GetInstanceDetails() (InstanceDetails, error) {
	result := i.InstanceDetails
	if result.ShootName == "" {
		logrus.Infof("extracting shoot name/domain from dashboard_url %s for instance %s", i.DashboardURL, i.InstanceID)
		shoot, domain, e := i.extractShootNameAndDomain()
		if e != nil {
			logrus.Errorf("unable to extract shoot name: %s (instance %s)", e.Error(), i.InstanceID)
			return result, e
		}
		result.ShootName = shoot
		result.ShootDomain = domain
	}
	//overwrite RuntimeID in InstanceDetails with Instance.RuntimeID
	//needed for runtimes suspended without clearing RuntimeID in deprovisioning operation
	result.RuntimeID = i.RuntimeID
	return result, nil
}

func (i *Instance) extractShootNameAndDomain() (string, string, error) {
	parsed, err := url.Parse(i.DashboardURL)
	if err != nil {
		return "", "", errors.Wrapf(err, "while parsing dashboard url %s", i.DashboardURL)
	}

	parts := strings.Split(parsed.Host, ".")
	if len(parts) <= 1 {
		return "", "", fmt.Errorf("host is too short: %s", parsed.Host)
	}
	return parts[1], parsed.Host[len(parts[0])+1:], nil
}

// OperationType defines the possible types of an asynchronous operation to a broker.
type OperationType string

const (
	// OperationTypeProvision means provisioning OperationType
	OperationTypeProvision OperationType = "provision"
	// OperationTypeDeprovision means deprovision OperationType
	OperationTypeDeprovision OperationType = "deprovision"
	// OperationTypeUndefined means undefined OperationType
	OperationTypeUndefined OperationType = ""
	// OperationTypeUpgradeKyma means upgrade Kyma OperationType
	OperationTypeUpgradeKyma OperationType = "upgradeKyma"
	// OperationTypeUpdate means update
	OperationTypeUpdate OperationType = "update"
	// OperationTypeUpgradeCluster means upgrade cluster (shoot) OperationType
	OperationTypeUpgradeCluster OperationType = "upgradeCluster"
)

type Operation struct {
	// following fields are serialized to JSON and stored in the storage
	InstanceDetails

	ID        string        `json:"-"`
	Version   int           `json:"-"`
	CreatedAt time.Time     `json:"-"`
	UpdatedAt time.Time     `json:"-"`
	Type      OperationType `json:"-"`

	InstanceID             string                    `json:"-"`
	ProvisionerOperationID string                    `json:"-"`
	State                  domain.LastOperationState `json:"-"`
	Description            string                    `json:"-"`
	ProvisioningParameters ProvisioningParameters    `json:"-"`

	// OrchestrationID specifies the origin orchestration which triggers the operation, empty for OSB operations (provisioning/deprovisioning)
	OrchestrationID string              `json:"-"`
	FinishedStages  map[string]struct{} `json:"-"`
}

func (o *Operation) IsFinished() bool {
	return o.State != orchestration.InProgress && o.State != orchestration.Pending && o.State != orchestration.Canceling
}

// Orchestration holds all information about an orchestration.
// Orchestration performs operations of a specific type (UpgradeKymaOperation, UpgradeClusterOperation)
// on specific targets of SKRs.
type Orchestration struct {
	OrchestrationID string
	Type            orchestration.Type
	State           string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Parameters      orchestration.Parameters
}

func (o *Orchestration) IsFinished() bool {
	return o.State == orchestration.Succeeded || o.State == orchestration.Failed || o.State == orchestration.Canceled
}

// IsCanceled returns true if orchestration's cancellation endpoint was ever triggered
func (o *Orchestration) IsCanceled() bool {
	return o.State == orchestration.Canceling || o.State == orchestration.Canceled
}

type InstanceWithOperation struct {
	Instance

	Type           sql.NullString
	State          sql.NullString
	Description    sql.NullString
	OpCreatedAt    time.Time
	IsSuspensionOp bool
}

type SMClientFactory interface {
	ForCredentials(credentials *servicemanager.Credentials) servicemanager.Client
	ForCustomerCredentials(request servicemanager.RequestContext, log logrus.FieldLogger) (servicemanager.Client, error)
	ProvideCredentials(request servicemanager.RequestContext, log logrus.FieldLogger) (*servicemanager.Credentials, error)
}

type InstanceDetails struct {
	Avs      AvsLifecycleData `json:"avs"`
	EventHub EventHub         `json:"eh"`

	SubAccountID string           `json:"sub_account_id"`
	RuntimeID    string           `json:"runtime_id"`
	ShootName    string           `json:"shoot_name"`
	ShootDomain  string           `json:"shoot_domain"`
	XSUAA        XSUAAData        `json:"xsuaa"`
	Ems          EmsData          `json:"ems"`
	Connectivity ConnectivityData `json:"connectivity"`
}

// ProvisioningOperation holds all information about provisioning operation
type ProvisioningOperation struct {
	Operation

	RuntimeVersion RuntimeVersionData `json:"runtime_version"`
	DashboardURL   string             `json:"dashboardURL"`

	// following fields are not stored in the storage
	InputCreator ProvisionerInputCreator `json:"-"`

	SMClientFactory SMClientFactory `json:"-"`
}

type ServiceManagerInstanceInfo struct {
	BrokerID                string `json:"brokerId"`
	ServiceID               string `json:"serviceId"`
	PlanID                  string `json:"planId"` // it is a plan.CatalogID from the Service Manager perspective
	InstanceID              string `json:"instanceId"`
	Provisioned             bool   `json:"provisioned"`
	ProvisioningTriggered   bool   `json:"provisioningTriggered"`
	DeprovisioningTriggered bool   `json:"deprovisioningTriggered"`
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

type ConnectivityData struct {
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

	// Temporary indicates that this deprovisioning operation must not remove the instance
	Temporary bool `json:"temporary"`
}

type UpdatingOperation struct {
	Operation

	UpdatingParameters UpdatingParametersDTO `json:"updating_parameters"`

	// following fields are not stored in the storage
	InputCreator ProvisionerInputCreator `json:"-"`
}

// UpgradeKymaOperation holds all information about upgrade Kyma operation
type UpgradeKymaOperation struct {
	Operation

	orchestration.RuntimeOperation `json:"runtime_operation"`
	InputCreator                   ProvisionerInputCreator `json:"-"`

	RuntimeVersion RuntimeVersionData `json:"runtime_version"`

	SMClientFactory SMClientFactory `json:"-"`
}

// UpgradeClusterOperation holds all information about upgrade cluster (shoot) operation
type UpgradeClusterOperation struct {
	Operation

	orchestration.RuntimeOperation `json:"runtime_operation"`
	InputCreator                   ProvisionerInputCreator `json:"-"`
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
			Type:                   OperationTypeProvision,
			ProvisioningParameters: parameters,
			InstanceDetails: InstanceDetails{
				SubAccountID: parameters.ErsContext.SubAccountID,
			},
			FinishedStages: make(map[string]struct{}, 0),
		},
	}, nil
}

// NewDeprovisioningOperationWithID creates a fresh (just starting) instance of the DeprovisioningOperation with provided ID
func NewDeprovisioningOperationWithID(operationID string, instance *Instance) (DeprovisioningOperation, error) {
	details, err := instance.GetInstanceDetails()
	if err != nil {
		return DeprovisioningOperation{}, err
	}
	return DeprovisioningOperation{
		Operation: Operation{
			ID:              operationID,
			Version:         0,
			Description:     "Operation created",
			InstanceID:      instance.InstanceID,
			State:           domain.InProgress,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Type:            OperationTypeDeprovision,
			InstanceDetails: details,
			FinishedStages:  make(map[string]struct{}, 0),
		},
	}, nil
}

func NewUpdateOperation(operationID string, instance *Instance, updatingParams UpdatingParametersDTO) UpdatingOperation {

	op := UpdatingOperation{
		Operation: Operation{
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instance.InstanceID,
			State:                  orchestration.Pending,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   OperationTypeUpdate,
			InstanceDetails:        instance.InstanceDetails,
			FinishedStages:         make(map[string]struct{}, 0),
			ProvisioningParameters: instance.Parameters,
		},
		UpdatingParameters: updatingParams,
	}

	if updatingParams.OIDC != nil {
		op.ProvisioningParameters.Parameters.OIDC = updatingParams.OIDC
	}

	if len(updatingParams.RuntimeAdministrators) != 0 {
		op.ProvisioningParameters.Parameters.RuntimeAdministrators = updatingParams.RuntimeAdministrators
	}

	return op
}

// NewSuspensionOperationWithID creates a fresh (just starting) instance of the DeprovisioningOperation which does not remove the instance.
func NewSuspensionOperationWithID(operationID string, instance *Instance) DeprovisioningOperation {
	return DeprovisioningOperation{
		Operation: Operation{
			ID:              operationID,
			Version:         0,
			Description:     "Operation created",
			InstanceID:      instance.InstanceID,
			State:           orchestration.Pending,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Type:            OperationTypeDeprovision,
			InstanceDetails: instance.InstanceDetails,
			FinishedStages:  make(map[string]struct{}, 0),
		},
		Temporary: true,
	}
}

func (po *ProvisioningOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	return po.SMClientFactory.ForCustomerCredentials(serviceManagerRequestCreds(po.ProvisioningParameters), log)
}

func (po *ProvisioningOperation) ProvideServiceManagerCredentials(log logrus.FieldLogger) (*servicemanager.Credentials, error) {
	return po.SMClientFactory.ProvideCredentials(serviceManagerRequestCreds(po.ProvisioningParameters), log)
}

func (o *Operation) FinishStage(stageName string) {
	o.FinishedStages[stageName] = struct{}{}
}

func (o *Operation) IsStageFinished(stage string) bool {
	_, found := o.FinishedStages[stage]
	return found
}

func (do *DeprovisioningOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	return do.SMClientFactory.ForCustomerCredentials(serviceManagerRequestCreds(do.ProvisioningParameters), log)
}

func (uko *UpgradeKymaOperation) ServiceManagerClient(log logrus.FieldLogger) (servicemanager.Client, error) {
	return uko.SMClientFactory.ForCustomerCredentials(serviceManagerRequestCreds(uko.ProvisioningParameters), log)
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

func serviceManagerRequestCreds(parameters ProvisioningParameters) servicemanager.RequestContext {
	var creds *servicemanager.Credentials
	sm := parameters.ErsContext.ServiceManager
	if sm != nil {
		creds = &servicemanager.Credentials{
			Username: sm.Credentials.BasicAuth.Username,
			Password: sm.Credentials.BasicAuth.Password,
			URL:      sm.URL,
		}
	}
	return servicemanager.RequestContext{
		SubaccountID: parameters.ErsContext.SubAccountID,
		Credentials:  creds,
	}
}

func (i *ServiceManagerInstanceInfo) ToProvisioningInput() *servicemanager.ProvisioningInput {
	var input servicemanager.ProvisioningInput

	input.ID = i.InstanceID
	input.ServiceID = i.ServiceID
	input.PlanID = i.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()

	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = map[string]interface{}{}

	return &input
}
