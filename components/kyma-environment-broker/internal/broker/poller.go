package broker

import (
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Poller interface {
	Invoke(logic func() error) error
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

func (p *DefaultPoller) Invoke(logic func() error) error {
	return wait.PollImmediate(p.PollInterval, p.PollTimeout, func() (bool, error) {
		err := logic()
		if err != nil {
			log.Warn(errors.Wrap(err, "while polling").Error())
			return false, nil
		}
		return true, nil
	})
}

type PassthroughPoller struct {
}

func NewPassthroughPoller() Poller {
	return &PassthroughPoller{}
}

func (p *PassthroughPoller) Invoke(logic func() error) error {
	return logic()
}
