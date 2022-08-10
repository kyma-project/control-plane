package handlers

import (
	"fmt"
	"time"

	commonOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Retryer struct {
	orchestrations storage.Orchestrations
	operations     storage.Operations
	queue          *process.Queue
	log            logrus.FieldLogger
}

type kymaRetryer Retryer

func NewKymaRetryer(orchestrations storage.Orchestrations, operations storage.Operations, q *process.Queue, logger logrus.FieldLogger) *kymaRetryer {
	return &kymaRetryer{
		orchestrations: orchestrations,
		operations:     operations,
		queue:          q,
		log:            logger,
	}
}

func (r *kymaRetryer) orchestrationRetry(o *internal.Orchestration, opsByOrch []internal.UpgradeKymaOperation, operationIDs []string, immediate string) (commonOrchestration.RetryResponse, error) {
	var err error
	resp := commonOrchestration.RetryResponse{OrchestrationID: o.OrchestrationID}

	ops, invalidIDs := r.orchestrationOperationsFilter(opsByOrch, operationIDs)
	resp.InvalidOperations = invalidIDs
	if len(ops) == 0 {
		zeroValidOperationInfo(&resp, r.log)
		return resp, nil
	}

	// as failed orchestration has finished before
	// only retry the latest failed kyma upgrade operation for the same instance
	if o.State == commonOrchestration.Failed {
		var oldIDs []string
		var err error

		ops, oldIDs, err = r.latestOperationValidate(o.OrchestrationID, ops)
		if err != nil {
			return resp, err
		}
		resp.OldOperations = oldIDs

		if len(ops) == 0 {
			zeroValidOperationInfo(&resp, r.log)
			return resp, nil
		}
	}

	for _, op := range ops {
		resp.RetryOperations = append(resp.RetryOperations, op.Operation.ID)
	}
	resp.Msg = "retry operations are queued for processing"

	err = r.OperationsStateUpdate(ops, immediate)
	if err != nil {
		return resp, err
	}

	// get orchestration state again in case in progress changed to failed, need to put in queue
	lastState, err := orchestrationStateUpdate(r.orchestrations, o.OrchestrationID, r.log)
	if err != nil {
		return resp, err
	}

	r.log.Infof("Converting orchestration %s from state %s to retrying", o.OrchestrationID, lastState)
	if lastState == commonOrchestration.Failed {
		r.queue.Add(o.OrchestrationID)
	}

	return resp, nil
}

// filter out the operation which doesn't belong to the given orchestration
func (r *kymaRetryer) orchestrationOperationsFilter(opsByOrch []internal.UpgradeKymaOperation, opsIDs []string) ([]internal.UpgradeKymaOperation, []string) {
	if len(opsIDs) <= 0 {
		return opsByOrch, nil
	}

	var retOps []internal.UpgradeKymaOperation
	var invalidIDs []string
	var found bool

	for _, opID := range opsIDs {
		for _, op := range opsByOrch {
			if opID == op.Operation.ID {
				retOps = append(retOps, op)
				found = true
				break
			}
		}

		if found {
			found = false
		} else {
			invalidIDs = append(invalidIDs, opID)
		}
	}

	return retOps, invalidIDs
}

// if the required operation for kyma upgrade is not the last operated operation for kyma upgrade, then report error
// only validate for failed orchestration
func (r *kymaRetryer) latestOperationValidate(orchestrationID string, ops []internal.UpgradeKymaOperation) ([]internal.UpgradeKymaOperation, []string, error) {
	var retryOps []internal.UpgradeKymaOperation
	var oldIDs []string

	for _, op := range ops {
		instanceID := op.InstanceID

		kymaOps, err := r.operations.ListUpgradeKymaOperationsByInstanceID(instanceID)
		if err != nil {
			// fail for listing operations of one instance, then http return and report fail
			r.log.Errorf("while getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "while getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		var errFound, newerExist bool
		num := len(kymaOps)

		for i := 0; i < num; i++ {
			if op.CreatedAt.Before(kymaOps[i].CreatedAt) {
				if num == 1 {
					errFound = true
					break
				}

				// 'canceled' or 'canceling' newer op is not a newer op
				if kymaOps[i].State == commonOrchestration.Canceled || kymaOps[i].State == commonOrchestration.Canceling {
					continue
				}

				oldIDs = append(oldIDs, op.Operation.ID)
				newerExist = true
			}

			break
		}

		if num == 0 || errFound {
			r.log.Errorf("while getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "while getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		if newerExist {
			continue
		}

		retryOps = append(retryOps, op)
	}

	return retryOps, oldIDs, nil
}

func (r *kymaRetryer) OperationsStateUpdate(ops []internal.UpgradeKymaOperation, immediate string) error {
	for _, op := range ops {
		if immediate == "true" {
			op.MaintenanceWindowBegin = time.Time{}
			op.MaintenanceWindowEnd = time.Time{}
		}
		op.State = commonOrchestration.Retrying
		op.UpdatedAt = time.Now()
		op.Description = "queued for retrying"

		_, err := r.operations.UpdateUpgradeKymaOperation(op)
		if err != nil {
			// one update fail then http return
			r.log.Errorf("Cannot update operation %s in storage: %s", op.Operation.ID, err)
			return errors.Wrapf(err, "while updating orchestration %s", op.OrchestrationID)
		}
	}

	return nil
}

func orchestrationStateUpdate(orchestrations storage.Orchestrations, orchestrationID string, log logrus.FieldLogger) (string, error) {
	o, err := orchestrations.GetByID(orchestrationID)
	if err != nil {
		log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		return "", errors.Wrapf(err, "while getting orchestration %s", orchestrationID)
	}
	// last minute check in case in progress one got canceled.
	state := o.State
	if state == commonOrchestration.Canceling || state == commonOrchestration.Canceled {
		log.Infof("orchestration %s was canceled right before retrying", orchestrationID)
		return state, fmt.Errorf("orchestration %s was canceled right before retrying", orchestrationID)
	}

	o.UpdatedAt = time.Now()
	if state == commonOrchestration.Failed {
		o.Description += ", retrying"
		o.State = commonOrchestration.Retrying
	}
	err = orchestrations.Update(*o)
	if err != nil {
		log.Errorf("while updating orchestration %s: %v", orchestrationID, err)
		return state, errors.Wrapf(err, "while updating orchestration %s", orchestrationID)
	}
	return state, nil
}

func zeroValidOperationInfo(resp *commonOrchestration.RetryResponse, log logrus.FieldLogger) {
	log.Infof("no valid operations to retry for orchestration %s", resp.OrchestrationID)
	resp.Msg = fmt.Sprintf("No valid operations to retry for orchestration %s", resp.OrchestrationID)
}
