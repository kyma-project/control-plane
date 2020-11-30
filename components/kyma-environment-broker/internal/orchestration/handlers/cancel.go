package handlers

import (
	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Canceler struct {
	orchestrations storage.Orchestrations
	operations     storage.Operations
	log            logrus.FieldLogger
}

func NewCanceler(operations storage.Operations, orchestrations storage.Orchestrations, logger logrus.FieldLogger) *Canceler {
	return &Canceler{
		orchestrations: orchestrations,
		operations:     operations,
		log:            logger,
	}
}

// Cancel finds in progress orchestration and cancels it
func (c *Canceler) Cancel() (string, error) {
	orchestrations, size, _, err := c.orchestrations.List(dbmodel.OrchestrationFilter{States: []string{orchestrationExt.InProgress}})
	if err != nil {
		return "", errors.Wrap(err, "while listing orchestrations")
	}
	if size == 0 {
		return "", dberr.NotFound("orchestration in progress was not found")
	}
	if size > 1 {
		ids := make([]string, 0)
		for _, o := range orchestrations {
			ids = append(ids, o.OrchestrationID)
		}
		return "", errors.Errorf("there should be only one in progress orchestration, found: %d, ids: %v", size, ids)
	}

	return orchestrations[0].OrchestrationID, c.CancelForID(orchestrations[0].OrchestrationID)
}

// CancelForID cancels orchestration by ID
func (c *Canceler) CancelForID(orchestrationID string) error {
	o, err := c.orchestrations.GetByID(orchestrationID)
	if err != nil {
		return errors.Wrap(err, "while getting orchestration")
	}
	o.State = orchestrationExt.Canceled
	err = c.orchestrations.Update(*o)
	if err != nil {
		return errors.Wrap(err, "while updating orchestration")
	}
	return nil
}
