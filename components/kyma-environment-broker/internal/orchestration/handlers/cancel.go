package handlers

import (
	"fmt"
	"time"

	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Canceler struct {
	orchestrations storage.Orchestrations
	log            logrus.FieldLogger
}

func NewCanceler(orchestrations storage.Orchestrations, logger logrus.FieldLogger) *Canceler {
	return &Canceler{
		orchestrations: orchestrations,
		log:            logger,
	}
}

// CancelForID cancels orchestration by ID
func (c *Canceler) CancelForID(orchestrationID string) error {
	o, err := c.orchestrations.GetByID(orchestrationID)
	if err != nil {
		return fmt.Errorf("while getting orchestration: %w", err)
	}
	if o.IsFinished() || o.State == orchestrationExt.Canceling {
		return nil
	}

	o.UpdatedAt = time.Now()
	o.Description = "Orchestration was canceled"
	o.State = orchestrationExt.Canceling
	err = c.orchestrations.Update(*o)
	if err != nil {
		return fmt.Errorf("while updating orchestration: %w", err)
	}
	return nil
}
