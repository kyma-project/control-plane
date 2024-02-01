package keb

import "time"

type Config struct {
	URL              string        `envconfig:"KEB_URL" required:"true"`
	Timeout          time.Duration `default:"30s"       envconfig:"KEB_TIMEOUT"`
	RetryCount       int           `default:"5"         envconfig:"KEB_RETRY_COUNT"`
	PollWaitDuration time.Duration `default:"10m"       envconfig:"KEB_POLL_WAIT_DURATION"`
}
