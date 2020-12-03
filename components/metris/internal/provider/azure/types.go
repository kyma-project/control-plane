package azure

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/kyma-project/control-plane/components/metris/internal/storage"
	"k8s.io/client-go/util/workqueue"
)

var (
	ErrResourceGroupNotFound  = errors.New("resource group not found")
	ErrMetricClient           = errors.New("metric client error")
	ErrMetricNotFound         = errors.New("no metric found")
	ErrTimeseriesNotFound     = errors.New("no timeseries found")
	ErrTimeseriesDataNotFound = errors.New("no timeserie data found")
)

const (
	// for more details on available capabilities, see https://docs.microsoft.com/en-ca/rest/api/compute/resourceskus/list.
	// capMemoryGB is the capability name for the memory in GB of a virtual machine.
	capMemoryGB string = "MemoryGB"
	// capvCPUs is the capability name for the virtual cpus of a virtual machine.
	capvCPUs string = "vCPUs"

	// tagNameSubAccountID is the subAccountID tag name use for tagging resource group.
	tagNameSubAccountID string = "SubAccountID"

	// diskSizeFactor is to calculate rounded value of disk size in gigabytes, example 17Gb->32Gb, 33Gb->64Gb.
	diskSizeFactor float64 = 32

	// intervalPT5M represent an interval of 5 minutes for the insight metric query.
	intervalPT5M time.Duration = 5 * time.Minute

	// PT1M ...
	PT1M TimeGrain = "PT1M"
	// PT5M ...
	PT5M TimeGrain = "PT5M"

	// maximum number of failed attempt to get metrics, after that instance is remove from cache/storage.
	maxRetryAttempts int = 5

	responseErrCodeResourceGroupNotFound string = "ResourceGroupNotFound"
)

// ResponseError represent the error message structure return by Azure REST API.
type ResponseError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TimeGrain enumerates the values for time grain.
type TimeGrain string

// vmCapabilities represent the available capabilities per vm size.
type vmCapabilities map[string]map[string]string

//go:generate mockery --name Client
// Client is the interface that provides the methods to get metrics from Azure Rest API.
type Client interface {
	GetResourceGroup(ctx context.Context, name, filter string, logger log.Logger) (resources.Group, error)
	GetVirtualMachines(ctx context.Context, rgname string, logger log.Logger) ([]compute.VirtualMachine, error)
	GetVMResourceSkus(ctx context.Context, filter string, logger log.Logger) ([]compute.ResourceSku, error)
	GetDisks(ctx context.Context, rgname string, logger log.Logger) ([]compute.Disk, error)
	GetLoadBalancers(ctx context.Context, rgname string, logger log.Logger) ([]network.LoadBalancer, error)
	GetVirtualNetworks(ctx context.Context, rgname string, logger log.Logger) ([]network.VirtualNetwork, error)
	GetPublicIPAddresses(ctx context.Context, rgname string, logger log.Logger) ([]network.PublicIPAddress, error)
	GetEHNamespaces(ctx context.Context, rgname string, logger log.Logger) ([]eventhub.EHNamespace, error)
	GetMetricValues(ctx context.Context, resourceURI, interval string, metricnames, aggregations []string, logger log.Logger) (map[string]insights.MetricValue, error)
}

// client holds Azure clients configuration.
type client struct {
	computeBaseClient   *compute.BaseClient
	networkBaseClient   *network.BaseClient
	insightsBaseClient  *insights.BaseClient
	resourcesBaseClient *resources.BaseClient
	eventhubBaseClient  *eventhub.BaseClient
}

var _ Client = (*client)(nil)

// baseClient represent the base configuration to use for all the REST API base client.
type baseClient struct {
	autorest.Client
	BaseURI        string
	SubscriptionID string
}

// Instance is an instance of a cluster with its client configuration.
type Instance struct {
	// cluster holds the gardener cluster information.
	cluster *gardener.Cluster
	// client holds the Azure base clients for the different API calls.
	client Client
	// lastEvent store the last successful event sent to EDP.
	lastEvent *EventData
	// eventHubResourceGroupName store the Azure Event Hub resource group name associated with the subaccountid.
	eventHubResourceGroupName string
	// retryAttempts store the number of retry attempts to get metrics.
	retryAttempts int
	// retryBackoff indicate to backing off between requests.
	retryBackoff bool
}

//go:generate mockery --name AuthConfig
// AuthConfig is the interface that provides an Authoriser used to provide request authorization to the Azure Rest API.
type AuthConfig interface {
	GetAuthConfig(clientID, clientSecret, tenantID, envName string) (autorest.Authorizer, azure.Environment, error)
}

// DefaultAuthConfig implements a default AuthConfig.
type DefaultAuthConfig struct{}

// Azure holds the Azure provider configuration options.
type Azure struct {
	config           *provider.Config
	instanceStorage  storage.Storage
	vmCapsStorage    storage.Storage
	queue            workqueue.DelayingInterface
	ClientAuthConfig AuthConfig
}

// ClientSecretMap is a structure to decode and map kubernetes secret data values to azure client configuration.
type ClientSecretMap struct {
	ClientID        string `mapstructure:"clientID"`
	ClientSecret    string `mapstructure:"clientSecret"`
	TenantID        string `mapstructure:"tenantID"`
	SubscriptionID  string `mapstructure:"subscriptionID"`
	EnvironmentName string
}

// VMType defines the event format for the virtual machine metrics.
type VMType struct {
	Name  string `json:"name"`
	Count uint32 `json:"count"`
}

// ProvisionedVolume defines the event format for the volume metrics.
type ProvisionedVolume struct {
	SizeGBTotal   uint32 `json:"size_gb_total"`
	SizeGBRounded uint32 `json:"size_gb_rounded"`
	Count         uint32 `json:"count"`
}

// Compute defines the event format for the compute metrics.
type Compute struct {
	VMTypes            []VMType          `json:"vm_types"`
	ProvisionedRAMGB   float64           `json:"provisioned_ram_gb"`
	ProvisionedVolumes ProvisionedVolume `json:"provisioned_volumes"`
	ProvisionedCpus    uint32            `json:"provisioned_cpus"`
}

// Networking defines the event format for the network metrics.
type Networking struct {
	ProvisionedLoadBalancers uint32 `json:"provisioned_loadbalancers"`
	ProvisionedVnets         uint32 `json:"provisioned_vnets"`
	ProvisionedIps           uint32 `json:"provisioned_ips"`
}

// EventHub defines the event format for the event hub metrics.
type EventHub struct {
	NumberNamespaces     uint32  `json:"number_namespaces"`
	IncomingRequestsPT1M float64 `json:"incoming_requests_pt1m"`
	MaxIncomingBytesPT1M float64 `json:"max_incoming_bytes_pt1m"`
	MaxOutgoingBytesPT1M float64 `json:"max_outgoing_bytes_pt1m"`
	IncomingRequestsPT5M float64 `json:"incoming_requests_pt5m"`
	MaxIncomingBytesPT5M float64 `json:"max_incoming_bytes_pt5m"`
	MaxOutgoingBytesPT5M float64 `json:"max_outgoing_bytes_pt5m"`
}

// EventData defines the event information to send to EDP.
type EventData struct {
	ResourceGroups []string    `json:"resource_groups"`
	Compute        *Compute    `json:"compute"`
	Networking     *Networking `json:"networking"`
	EventHub       *EventHub   `json:"event_hub"`
}
