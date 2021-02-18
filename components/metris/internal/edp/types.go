package edp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/log"

	"k8s.io/client-go/util/workqueue"
)

// Config holds EDP clients configuration.
type Config struct {
	URL               string        `kong:"help='EDP base URL',env='EDP_URL',default='https://input.yevents.io',required=true"`
	Token             string        `kong:"help='EDP source token',placeholder='SECRET',env='EDP_TOKEN',required=true"`
	Namespace         string        `kong:"help='EDP Namespace',env='EDP_NAMESPACE',required=true"`
	DataStream        string        `kong:"help='EDP data stream name',env='EDP_DATASTREAM_NAME',required=true"`
	DataStreamVersion string        `kong:"help='EDP data stream version',env='EDP_DATASTREAM_VERSION',required=true"`
	DataStreamEnv     string        `kong:"help='EDP data stream environment',env='EDP_DATASTREAM_ENV',required=true"`
	Timeout           time.Duration `kong:"help='Time limit for requests made by the EDP client',env='EDP_TIMEOUT',required=true,default='30s'"`
	Buffer            int           `kong:"help='Number of events that the buffer can have.',env='EDP_BUFFER',required=true,default=100"`
	Workers           int           `kong:"help='Number of workers to send metrics.',env='EDP_WORKERS',required=true,default=5"`
	EventRetry        int           `kong:"help='Number of retries for sending event.',env='EDP_RETRY',required=true,default=5"`
}

// Client has all the context and parameters needed to run a EDP worker pool.
type Client struct {
	// config represent the EDP client configuration.
	config *Config
	// httpClient define the EDP HTTP client.
	httpClient *http.Client
	// logger is the standard logger for the client.
	logger log.Logger
	// queue holds the events to send to EDP with rate limiting for failed sent event.
	queue workqueue.RateLimitingInterface
	// eventsChannel define the channel to exchange events with the provider.
	eventsChannel <-chan *Event
}

// Event has the information needed to send an event to EDP.
type Event struct {
	// Datatenant defines the tenant the event belongs to.
	Datatenant string

	// Data represent the provider specific event details in raw json, to delay json decoding.
	Data *json.RawMessage
}

// MarshalJSON is a helper function to marshal custom provider data and add timestamp field.
func (e Event) MarshalJSON() ([]byte, error) {
	ts := time.Now().Format(time.RFC3339)

	event, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`{"timestamp": "%s", %s`, ts, event[1:])), nil
}
