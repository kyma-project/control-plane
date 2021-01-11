package provider

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
)

// Factory generates a Provider.
type Factory func(config *Config) Provider

// Config holds providers base configuration.
type Config struct {
	PollInterval    time.Duration `kong:"help='Interval at which metrics are fetch.',env='PROVIDER_POLLINTERVAL',required=false,default='5m'"`
	PollMaxInterval time.Duration `kong:"help='maximum Interval at which metrics are fetch.',env='PROVIDER_POLLMAXINTERVAL',required=false,default='15m'"`
	PollingDuration time.Duration `kong:"help='Time limit for requests made by the provider client.',env='PROVIDER_POLLDURATION',required=false,default='5m'"`
	Workers         int           `kong:"help='Number of workers to fetch metrics.',env='PROVIDER_WORKERS',required=true,default=10"`
	Buffer          int           `kong:"help='Number of cluster that the buffer can have.',env='PROVIDER_BUFFER',required=true,default=100"`
	MaxRetries      int           `kong:"help='Maximum number of retries before a cluster is removed from the cache if it is not found on the provider. NOTE: This will stop sending events for the removed cluster',env='PROVIDER_MAXRETRIES',required=true,default=20"`

	// ClusterChannel define the channel to exchange clusters information with Gardener controller.
	ClusterChannel chan *gardener.Cluster `kong:"-"`
	// EventsChannel define the channel to exchange events with EDP.
	EventsChannel chan<- *edp.Event `kong:"-"`
	// logger is the standard logger for the provider.
	Logger log.Logger `kong:"-"`
}

// Provider interface contains all behaviors for a provider.
type Provider interface {
	Run(ctx context.Context)
}

// SecretMap is the interface that provides a method to decode kubenertes secrets into a Provider custom structure.
type SecretMap interface {
	decode(secrets map[string][]byte) error
}
