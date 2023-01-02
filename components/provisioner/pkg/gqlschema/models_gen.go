// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package gqlschema

import (
	"fmt"
	"io"
	"strconv"
)

type ProviderSpecificConfig interface {
	IsProviderSpecificConfig()
}

type AWSProviderConfig struct {
	AwsZones []*AWSZone `json:"awsZones"`
	VpcCidr  *string    `json:"vpcCidr"`
}

func (AWSProviderConfig) IsProviderSpecificConfig() {}

type AWSProviderConfigInput struct {
	VpcCidr  string          `json:"vpcCidr"`
	AwsZones []*AWSZoneInput `json:"awsZones"`
}

type AWSZone struct {
	Name         *string `json:"name"`
	PublicCidr   *string `json:"publicCidr"`
	InternalCidr *string `json:"internalCidr"`
	WorkerCidr   *string `json:"workerCidr"`
}

type AWSZoneInput struct {
	Name         string `json:"name"`
	PublicCidr   string `json:"publicCidr"`
	InternalCidr string `json:"internalCidr"`
	WorkerCidr   string `json:"workerCidr"`
}

type AzureProviderConfig struct {
	VnetCidr                     *string      `json:"vnetCidr"`
	Zones                        []string     `json:"zones"`
	AzureZones                   []*AzureZone `json:"azureZones"`
	EnableNatGateway             *bool        `json:"enableNatGateway"`
	IdleConnectionTimeoutMinutes *int         `json:"idleConnectionTimeoutMinutes"`
}

func (AzureProviderConfig) IsProviderSpecificConfig() {}

type AzureProviderConfigInput struct {
	VnetCidr                     string            `json:"vnetCidr"`
	Zones                        []string          `json:"zones"`
	AzureZones                   []*AzureZoneInput `json:"azureZones"`
	EnableNatGateway             *bool             `json:"enableNatGateway"`
	IdleConnectionTimeoutMinutes *int              `json:"idleConnectionTimeoutMinutes"`
}

type AzureZone struct {
	Name int    `json:"name"`
	Cidr string `json:"cidr"`
}

type AzureZoneInput struct {
	Name int    `json:"name"`
	Cidr string `json:"cidr"`
}

type ClusterConfigInput struct {
	GardenerConfig *GardenerConfigInput `json:"gardenerConfig"`
	Administrators []string             `json:"administrators"`
}

type ComponentConfiguration struct {
	Component     string         `json:"component"`
	Namespace     string         `json:"namespace"`
	Configuration []*ConfigEntry `json:"configuration"`
	SourceURL     *string        `json:"sourceURL"`
}

type ComponentConfigurationInput struct {
	Component        string              `json:"component"`
	Namespace        string              `json:"namespace"`
	Configuration    []*ConfigEntryInput `json:"configuration"`
	SourceURL        *string             `json:"sourceURL"`
	ConflictStrategy *ConflictStrategy   `json:"conflictStrategy"`
}

type ConfigEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret *bool  `json:"secret"`
}

type ConfigEntryInput struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret *bool  `json:"secret"`
}

type DNSConfig struct {
	Domain    string         `json:"domain"`
	Providers []*DNSProvider `json:"providers"`
}

type DNSConfigInput struct {
	Domain    string              `json:"domain"`
	Providers []*DNSProviderInput `json:"providers"`
}

type DNSProvider struct {
	DomainsInclude []string `json:"domainsInclude"`
	Primary        bool     `json:"primary"`
	SecretName     string   `json:"secretName"`
	Type           string   `json:"type"`
}

type DNSProviderInput struct {
	DomainsInclude []string `json:"domainsInclude"`
	Primary        bool     `json:"primary"`
	SecretName     string   `json:"secretName"`
	Type           string   `json:"type"`
}

type Error struct {
	Message *string `json:"message"`
}

type GCPProviderConfig struct {
	Zones []string `json:"zones"`
}

func (GCPProviderConfig) IsProviderSpecificConfig() {}

type GCPProviderConfigInput struct {
	Zones []string `json:"zones"`
}

type GardenerConfig struct {
	Name                                *string                `json:"name"`
	KubernetesVersion                   *string                `json:"kubernetesVersion"`
	TargetSecret                        *string                `json:"targetSecret"`
	Provider                            *string                `json:"provider"`
	Region                              *string                `json:"region"`
	Seed                                *string                `json:"seed"`
	MachineType                         *string                `json:"machineType"`
	MachineImage                        *string                `json:"machineImage"`
	MachineImageVersion                 *string                `json:"machineImageVersion"`
	DiskType                            *string                `json:"diskType"`
	VolumeSizeGb                        *int                   `json:"volumeSizeGB"`
	WorkerCidr                          *string                `json:"workerCidr"`
	AutoScalerMin                       *int                   `json:"autoScalerMin"`
	AutoScalerMax                       *int                   `json:"autoScalerMax"`
	MaxSurge                            *int                   `json:"maxSurge"`
	MaxUnavailable                      *int                   `json:"maxUnavailable"`
	Purpose                             *string                `json:"purpose"`
	LicenceType                         *string                `json:"licenceType"`
	EnableKubernetesVersionAutoUpdate   *bool                  `json:"enableKubernetesVersionAutoUpdate"`
	EnableMachineImageVersionAutoUpdate *bool                  `json:"enableMachineImageVersionAutoUpdate"`
	AllowPrivilegedContainers           *bool                  `json:"allowPrivilegedContainers"`
	ProviderSpecificConfig              ProviderSpecificConfig `json:"providerSpecificConfig"`
	DNSConfig                           *DNSConfig             `json:"dnsConfig"`
	OidcConfig                          *OIDCConfig            `json:"oidcConfig"`
	ExposureClassName                   *string                `json:"exposureClassName"`
	ShootNetworkingFilterDisabled       *bool                  `json:"shootNetworkingFilterDisabled"`
	ControlPlaneFailureTolerance        *string                `json:"controlPlaneFailureTolerance"`
	EuAccess                            *bool                  `json:"euAccess"`
}

type GardenerConfigInput struct {
	Name                                string                 `json:"name"`
	KubernetesVersion                   string                 `json:"kubernetesVersion"`
	Provider                            string                 `json:"provider"`
	TargetSecret                        string                 `json:"targetSecret"`
	Region                              string                 `json:"region"`
	MachineType                         string                 `json:"machineType"`
	MachineImage                        *string                `json:"machineImage"`
	MachineImageVersion                 *string                `json:"machineImageVersion"`
	DiskType                            *string                `json:"diskType"`
	VolumeSizeGb                        *int                   `json:"volumeSizeGB"`
	WorkerCidr                          string                 `json:"workerCidr"`
	AutoScalerMin                       int                    `json:"autoScalerMin"`
	AutoScalerMax                       int                    `json:"autoScalerMax"`
	MaxSurge                            int                    `json:"maxSurge"`
	MaxUnavailable                      int                    `json:"maxUnavailable"`
	Purpose                             *string                `json:"purpose"`
	LicenceType                         *string                `json:"licenceType"`
	EnableKubernetesVersionAutoUpdate   *bool                  `json:"enableKubernetesVersionAutoUpdate"`
	EnableMachineImageVersionAutoUpdate *bool                  `json:"enableMachineImageVersionAutoUpdate"`
	AllowPrivilegedContainers           *bool                  `json:"allowPrivilegedContainers"`
	ProviderSpecificConfig              *ProviderSpecificInput `json:"providerSpecificConfig"`
	DNSConfig                           *DNSConfigInput        `json:"dnsConfig"`
	Seed                                *string                `json:"seed"`
	OidcConfig                          *OIDCConfigInput       `json:"oidcConfig"`
	ExposureClassName                   *string                `json:"exposureClassName"`
	ShootNetworkingFilterDisabled       *bool                  `json:"shootNetworkingFilterDisabled"`
	ControlPlaneFailureTolerance        *string                `json:"controlPlaneFailureTolerance"`
	EuAccess                            *bool                  `json:"euAccess"`
}

type GardenerUpgradeInput struct {
	KubernetesVersion                   *string                `json:"kubernetesVersion"`
	MachineType                         *string                `json:"machineType"`
	DiskType                            *string                `json:"diskType"`
	VolumeSizeGb                        *int                   `json:"volumeSizeGB"`
	AutoScalerMin                       *int                   `json:"autoScalerMin"`
	AutoScalerMax                       *int                   `json:"autoScalerMax"`
	MachineImage                        *string                `json:"machineImage"`
	MachineImageVersion                 *string                `json:"machineImageVersion"`
	MaxSurge                            *int                   `json:"maxSurge"`
	MaxUnavailable                      *int                   `json:"maxUnavailable"`
	Purpose                             *string                `json:"purpose"`
	EnableKubernetesVersionAutoUpdate   *bool                  `json:"enableKubernetesVersionAutoUpdate"`
	EnableMachineImageVersionAutoUpdate *bool                  `json:"enableMachineImageVersionAutoUpdate"`
	ProviderSpecificConfig              *ProviderSpecificInput `json:"providerSpecificConfig"`
	OidcConfig                          *OIDCConfigInput       `json:"oidcConfig"`
	ExposureClassName                   *string                `json:"exposureClassName"`
	ShootNetworkingFilterDisabled       *bool                  `json:"shootNetworkingFilterDisabled"`
}

type HibernationStatus struct {
	Hibernated          *bool `json:"hibernated"`
	HibernationPossible *bool `json:"hibernationPossible"`
}

type KymaConfig struct {
	Version       *string                   `json:"version"`
	Profile       *KymaProfile              `json:"profile"`
	Components    []*ComponentConfiguration `json:"components"`
	Configuration []*ConfigEntry            `json:"configuration"`
}

type KymaConfigInput struct {
	Version          string                         `json:"version"`
	Profile          *KymaProfile                   `json:"profile"`
	Components       []*ComponentConfigurationInput `json:"components"`
	Configuration    []*ConfigEntryInput            `json:"configuration"`
	ConflictStrategy *ConflictStrategy              `json:"conflictStrategy"`
}

type LastError struct {
	ErrMessage string `json:"errMessage"`
	Reason     string `json:"reason"`
	Component  string `json:"component"`
}

type OIDCConfig struct {
	ClientID       string   `json:"clientID"`
	GroupsClaim    string   `json:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix"`
}

type OIDCConfigInput struct {
	ClientID       string   `json:"clientID"`
	GroupsClaim    string   `json:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix"`
}

type OpenStackProviderConfig struct {
	Zones                []string `json:"zones"`
	FloatingPoolName     string   `json:"floatingPoolName"`
	CloudProfileName     string   `json:"cloudProfileName"`
	LoadBalancerProvider string   `json:"loadBalancerProvider"`
}

func (OpenStackProviderConfig) IsProviderSpecificConfig() {}

type OpenStackProviderConfigInput struct {
	Zones                []string `json:"zones"`
	FloatingPoolName     string   `json:"floatingPoolName"`
	CloudProfileName     string   `json:"cloudProfileName"`
	LoadBalancerProvider string   `json:"loadBalancerProvider"`
}

type OperationStatus struct {
	ID        *string        `json:"id"`
	Operation OperationType  `json:"operation"`
	State     OperationState `json:"state"`
	Message   *string        `json:"message"`
	RuntimeID *string        `json:"runtimeID"`
	LastError *LastError     `json:"lastError"`
}

type ProviderSpecificInput struct {
	GcpConfig       *GCPProviderConfigInput       `json:"gcpConfig"`
	AzureConfig     *AzureProviderConfigInput     `json:"azureConfig"`
	AwsConfig       *AWSProviderConfigInput       `json:"awsConfig"`
	OpenStackConfig *OpenStackProviderConfigInput `json:"openStackConfig"`
}

type ProvisionRuntimeInput struct {
	RuntimeInput  *RuntimeInput       `json:"runtimeInput"`
	ClusterConfig *ClusterConfigInput `json:"clusterConfig"`
	KymaConfig    *KymaConfigInput    `json:"kymaConfig"`
}

type RuntimeConfig struct {
	ClusterConfig *GardenerConfig `json:"clusterConfig"`
	KymaConfig    *KymaConfig     `json:"kymaConfig"`
	Kubeconfig    *string         `json:"kubeconfig"`
}

type RuntimeConnectionStatus struct {
	Status RuntimeAgentConnectionStatus `json:"status"`
	Errors []*Error                     `json:"errors"`
}

type RuntimeInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Labels      Labels  `json:"labels"`
}

type RuntimeStatus struct {
	LastOperationStatus     *OperationStatus         `json:"lastOperationStatus"`
	RuntimeConnectionStatus *RuntimeConnectionStatus `json:"runtimeConnectionStatus"`
	RuntimeConfiguration    *RuntimeConfig           `json:"runtimeConfiguration"`
	HibernationStatus       *HibernationStatus       `json:"hibernationStatus"`
}

type UpgradeRuntimeInput struct {
	KymaConfig *KymaConfigInput `json:"kymaConfig"`
}

type UpgradeShootInput struct {
	GardenerConfig *GardenerUpgradeInput `json:"gardenerConfig"`
	Administrators []string              `json:"administrators"`
}

type ConflictStrategy string

const (
	ConflictStrategyMerge   ConflictStrategy = "Merge"
	ConflictStrategyReplace ConflictStrategy = "Replace"
)

var AllConflictStrategy = []ConflictStrategy{
	ConflictStrategyMerge,
	ConflictStrategyReplace,
}

func (e ConflictStrategy) IsValid() bool {
	switch e {
	case ConflictStrategyMerge, ConflictStrategyReplace:
		return true
	}
	return false
}

func (e ConflictStrategy) String() string {
	return string(e)
}

func (e *ConflictStrategy) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ConflictStrategy(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ConflictStrategy", str)
	}
	return nil
}

func (e ConflictStrategy) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type KymaProfile string

const (
	KymaProfileEvaluation KymaProfile = "Evaluation"
	KymaProfileProduction KymaProfile = "Production"
)

var AllKymaProfile = []KymaProfile{
	KymaProfileEvaluation,
	KymaProfileProduction,
}

func (e KymaProfile) IsValid() bool {
	switch e {
	case KymaProfileEvaluation, KymaProfileProduction:
		return true
	}
	return false
}

func (e KymaProfile) String() string {
	return string(e)
}

func (e *KymaProfile) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = KymaProfile(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid KymaProfile", str)
	}
	return nil
}

func (e KymaProfile) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type OperationState string

const (
	OperationStatePending    OperationState = "Pending"
	OperationStateInProgress OperationState = "InProgress"
	OperationStateSucceeded  OperationState = "Succeeded"
	OperationStateFailed     OperationState = "Failed"
)

var AllOperationState = []OperationState{
	OperationStatePending,
	OperationStateInProgress,
	OperationStateSucceeded,
	OperationStateFailed,
}

func (e OperationState) IsValid() bool {
	switch e {
	case OperationStatePending, OperationStateInProgress, OperationStateSucceeded, OperationStateFailed:
		return true
	}
	return false
}

func (e OperationState) String() string {
	return string(e)
}

func (e *OperationState) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = OperationState(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid OperationState", str)
	}
	return nil
}

func (e OperationState) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type OperationType string

const (
	OperationTypeProvision            OperationType = "Provision"
	OperationTypeProvisionNoInstall   OperationType = "ProvisionNoInstall"
	OperationTypeUpgrade              OperationType = "Upgrade"
	OperationTypeUpgradeShoot         OperationType = "UpgradeShoot"
	OperationTypeDeprovision          OperationType = "Deprovision"
	OperationTypeDeprovisionNoInstall OperationType = "DeprovisionNoInstall"
	OperationTypeReconnectRuntime     OperationType = "ReconnectRuntime"
	OperationTypeHibernate            OperationType = "Hibernate"
)

var AllOperationType = []OperationType{
	OperationTypeProvision,
	OperationTypeProvisionNoInstall,
	OperationTypeUpgrade,
	OperationTypeUpgradeShoot,
	OperationTypeDeprovision,
	OperationTypeDeprovisionNoInstall,
	OperationTypeReconnectRuntime,
	OperationTypeHibernate,
}

func (e OperationType) IsValid() bool {
	switch e {
	case OperationTypeProvision, OperationTypeProvisionNoInstall, OperationTypeUpgrade, OperationTypeUpgradeShoot, OperationTypeDeprovision, OperationTypeDeprovisionNoInstall, OperationTypeReconnectRuntime, OperationTypeHibernate:
		return true
	}
	return false
}

func (e OperationType) String() string {
	return string(e)
}

func (e *OperationType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = OperationType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid OperationType", str)
	}
	return nil
}

func (e OperationType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type RuntimeAgentConnectionStatus string

const (
	RuntimeAgentConnectionStatusPending      RuntimeAgentConnectionStatus = "Pending"
	RuntimeAgentConnectionStatusConnected    RuntimeAgentConnectionStatus = "Connected"
	RuntimeAgentConnectionStatusDisconnected RuntimeAgentConnectionStatus = "Disconnected"
)

var AllRuntimeAgentConnectionStatus = []RuntimeAgentConnectionStatus{
	RuntimeAgentConnectionStatusPending,
	RuntimeAgentConnectionStatusConnected,
	RuntimeAgentConnectionStatusDisconnected,
}

func (e RuntimeAgentConnectionStatus) IsValid() bool {
	switch e {
	case RuntimeAgentConnectionStatusPending, RuntimeAgentConnectionStatusConnected, RuntimeAgentConnectionStatusDisconnected:
		return true
	}
	return false
}

func (e RuntimeAgentConnectionStatus) String() string {
	return string(e)
}

func (e *RuntimeAgentConnectionStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = RuntimeAgentConnectionStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid RuntimeAgentConnectionStatus", str)
	}
	return nil
}

func (e RuntimeAgentConnectionStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
