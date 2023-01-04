package broker

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

type Poller interface {
	Invoke(logic func() (bool, error)) error
}

type DefaultPoller struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

func NewDefaultPoller() Poller {
	return &DefaultPoller{
		PollInterval: 2 * time.Second,
		PollTimeout:  5 * time.Second,
	}
}

func (p *DefaultPoller) Invoke(logic func() (bool, error)) error {
	return wait.PollImmediate(p.PollInterval, p.PollTimeout, func() (bool, error) {
		return logic()
	})
}

type PassthroughPoller struct {
}

func NewPassthroughPoller() Poller {
	return &PassthroughPoller{}
}

func (p *PassthroughPoller) Invoke(logic func() (bool, error)) error {
	success, err := logic()
	if !success && err == nil {
		return fmt.Errorf("unsuccessful poll logic invocation")
	}
	return err
}
