package edp

import "time"

type Config struct {
	URL               string        `default:"https://input.yevents.io" envconfig:"EDP_URL"                required:"true"`
	Namespace         string        `default:"kyma-dev"                 envconfig:"EDP_NAMESPACE"          required:"true"`
	DataStreamName    string        `default:"consumption-metrics"      envconfig:"EDP_DATASTREAM_NAME"    required:"true"`
	DataStreamVersion string        `default:"1"                        envconfig:"EDP_DATASTREAM_VERSION" required:"true"`
	DataStreamEnv     string        `default:"dev"                      envconfig:"EDP_DATASTREAM_ENV"     required:"true"`
	Timeout           time.Duration `default:"30s"                      envconfig:"EDP_TIMEOUT"`
	EventRetry        int           `default:"3"                        envconfig:"EDP_RETRY"`
	Token             string
}
