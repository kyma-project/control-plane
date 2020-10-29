package postsql

import "time"

const (
	defaultRetryTimeout  = time.Second * 5
	defaultRetryInterval = time.Millisecond * 500
)
