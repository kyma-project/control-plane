package keb

import "time"

type Config struct {
	URL              string        `envconfig:"KEB_URL" required:"true"`
	Timeout          time.Duration `envconfig:"KEB_TIMEOUT" default:"30s"`
	RetryCount       int           `envconfig:"KEB_RETRY_COUNT" default:"5"`
	PollWaitDuration time.Duration `envconfig:"KEB_POLL_WAIT_DURATION" default:"10m"`
}
