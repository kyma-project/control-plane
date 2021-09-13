package broker

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

// Config holds configuration for the whole application
type KEBConfig struct {
	// DbInMemory allows to use memory storage instead of the postgres one.
	// Suitable for development purposes.
	DbInMemory bool `envconfig:"default=false"`

	// DisableProcessOperationsInProgress allows to disable processing operations
	// which are in progress on starting application. Set to true if you are
	// running in a separate testing deployment but with the production DB.
	DisableProcessOperationsInProgress bool `envconfig:"default=false"`

	// DevelopmentMode if set to true then errors are returned in http
	// responses, otherwise errors are only logged and generic message
	// is returned to client.
	// Currently works only with /info endpoints.
	DevelopmentMode bool `envconfig:"default=false"`

	// DumpProvisionerRequests enables dumping Provisioner requests. Must be disabled on Production environments
	// because some data must not be visible in the log file.
	DumpProvisionerRequests bool `envconfig:"default=false"`

	// OperationTimeout is used to check on a top-level if any operation didn't exceed the time for processing.
	// It is used for provisioning and deprovisioning operations.
	OperationTimeout time.Duration `envconfig:"default=24h"`

	Host       string `envconfig:"optional"`
	Port       string `envconfig:"default=8080"`
	StatusPort string `envconfig:"default=8071"`

	Provisioner input.Config
	Director    director.Config
	Database    storage.Config
	Gardener    gardener.Config
	Kubeconfig  kubeconfig.Config

	ServiceManager servicemanager.Config

	KymaVersion                          string
	KymaPreviewVersion                   string
	EnableOnDemandVersion                bool `envconfig:"default=false"`
	ManagedRuntimeComponentsYAMLFilePath string
	SkrOidcDefaultValuesYAMLFilePath     string
	DefaultRequestRegion                 string `envconfig:"default=cf-eu10"`
	UpdateProcessingEnabled              bool   `envconfig:"default=false"`

	Broker          Config
	CatalogFilePath string

	Avs avs.Config
	IAS ias.Config
	EDP edp.Config

	// Service Manager services
	XSUAA struct {
		Disabled bool `envconfig:"default=true"`
	}
	Connectivity struct {
		Disabled bool `envconfig:"default=true"`
	}

	AuditLog auditlog.Config

	VersionConfig struct {
		Namespace string
		Name      string
	}

	OrchestrationConfig struct {
		Namespace string
		Name      string
	}

	TrialRegionMappingFilePath string
	MaxPaginationPage          int `envconfig:"default=100"`

	LogLevel string `envconfig:"default=info"`

	// FreemiumProviders is a list of providers for freemium
	FreemiumProviders []string `envconfig:"default=aws"`

	DomainName string
}
