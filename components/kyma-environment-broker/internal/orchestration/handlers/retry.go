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
	resp           *commonOrchestration.RetryResponse
	log            logrus.FieldLogger
}

func NewRetryer(orchestrations storage.Orchestrations,
	operations storage.Operations,
	queue *process.Queue,
	logger logrus.FieldLogger) *Retryer {
	return &Retryer{
		orchestrations: orchestrations,
		operations:     operations,
		queue:          queue,
		resp:           &commonOrchestration.RetryResponse{},
		log:            logger,
	}
}

func (r *Retryer) stateUpdateForOrchestration(orchestrationID string) (string, error) {
	o, err := r.orchestrations.GetByID(orchestrationID)
	if err != nil {
		r.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		return o.State, errors.Wrapf(err, "while getting orchestration %s", orchestrationID)
	}
	// last minute check in case in progress one got canceled.
	if o.State == commonOrchestration.Canceling || o.State == commonOrchestration.Canceled {
		r.log.Infof("orchestration %s was canceled right before retrying", orchestrationID)
		return o.State, fmt.Errorf("orchestration %s was canceled right before retrying", orchestrationID)
	}

	o.UpdatedAt = time.Now()
	if o.State == commonOrchestration.Failed {
		o.Description = "queued for retrying"
		o.State = commonOrchestration.Retrying
	}
	err = r.orchestrations.Update(*o)
	if err != nil {
		r.log.Errorf("while updating orchestration %s: %v", orchestrationID, err)
		return o.State, errors.Wrapf(err, "while updating orchestration %s", orchestrationID)
	}
	return o.State, nil
}

func (r *Retryer) kymaUpgradeRetry(o *internal.Orchestration, opsByOrch []internal.UpgradeKymaOperation, operationIDs []string) error {
	r.resp.OrchestrationID = o.OrchestrationID

	ops, invalidIDs := r.kymaOrchestrationOperationsFilter(opsByOrch, operationIDs)
	r.resp.InvalidOperations = invalidIDs
	if len(ops) == 0 {
		r.resp.Msg = fmt.Sprintf("no valid operations to retry for orchestration %s", o.OrchestrationID)
		return nil
	}

	// as failed orchestration has finished before
	// only retry the latest failed kyma upgrade operation for the same instance
	if o.State == commonOrchestration.Failed {
		var oldIDs []string
		var err error

		ops, oldIDs, err = r.latestUpgradeKymaOperationValidate(o.OrchestrationID, ops)
		if err != nil {
			return err
		}
		r.resp.OldOperations = oldIDs

		if len(ops) == 0 {
			r.resp.Msg = fmt.Sprintf("no valid operations to retry for orchestration %s", o.OrchestrationID)
			return nil
		}
	}

	for _, op := range ops {
		r.resp.RetryOperations = append(r.resp.RetryOperations, op.Operation.ID)
	}

	err := r.stateUpdateForKymaUpgradeOperations(ops)
	if err != nil {
		return err
	}

	// get orchestration state again in case in progress changed to failed, need to put in queue
	lastState, err := r.stateUpdateForOrchestration(o.OrchestrationID)
	if err != nil {
		return err
	}

	if lastState == commonOrchestration.Failed {
		r.queue.Add(o.OrchestrationID)
	}

	return nil
}

func (r *Retryer) clusterUpgradeRetry(o *internal.Orchestration, opsByOrch []internal.UpgradeClusterOperation, operationIDs []string) error {
	r.resp.OrchestrationID = o.OrchestrationID

	ops, invalidIDs := r.clusterOrchestrationOperationsFilter(opsByOrch, operationIDs)
	r.resp.InvalidOperations = invalidIDs
	if len(ops) == 0 {
		r.resp.Msg = fmt.Sprintf("no valid operations to retry for orchestration %s", o.OrchestrationID)
		return nil
	}

	if o.State == commonOrchestration.Failed {
		var oldIDs []string
		var err error

		ops, oldIDs, err = r.latestUpgradeClusterOperationValidate(o.OrchestrationID, ops)
		if err != nil {
			return err
		}
		r.resp.OldOperations = oldIDs

		if len(ops) == 0 {
			r.resp.Msg = fmt.Sprintf("no valid operations to retry for orchestration %s", o.OrchestrationID)
			return nil
		}
	}

	for _, op := range ops {
		r.resp.RetryOperations = append(r.resp.RetryOperations, op.Operation.ID)
	}

	err := r.stateUpdateForClusterUpgradeOperations(ops)
	if err != nil {
		return err
	}

	// get orchestration state again in case in progress changed to failed, need to put in queue
	lastState, err := r.stateUpdateForOrchestration(o.OrchestrationID)
	if err != nil {
		return err
	}

	if lastState == commonOrchestration.Failed {
		r.queue.Add(o.OrchestrationID)
	}

	return nil
}

func (r *Retryer) stateUpdateForKymaUpgradeOperations(ops []internal.UpgradeKymaOperation) error {
	for _, op := range ops {
		op.State = commonOrchestration.Retrying
		op.UpdatedAt = time.Now()
		op.Description = "queued for retrying"

		_, err := r.operations.UpdateUpgradeKymaOperation(op)
		if err != nil {
			// one update fail then http return
			r.log.Errorf("Cannot update operation %s in storage: %s", op.Operation.ID, err)
			return errors.Wrapf(err, "while updating orchestration %s", r.resp.OrchestrationID)
		}
	}

	return nil
}

func (r *Retryer) stateUpdateForClusterUpgradeOperations(ops []internal.UpgradeClusterOperation) error {
	for _, op := range ops {
		op.State = commonOrchestration.Retrying
		op.UpdatedAt = time.Now()
		op.Description = "queued for retrying"

		_, err := r.operations.UpdateUpgradeClusterOperation(op)
		if err != nil {
			// one update fail then http return
			r.log.Errorf("Cannot update operation %s in storage: %s", op.Operation.ID, err)
			return errors.Wrapf(err, "while updating orchestration %s", r.resp.OrchestrationID)
		}
	}

	return nil
}

// if the required operation for kyma upgrade is not the last operated operation for kyma upgrade, then report error
// only validate for failed orchestration
func (r *Retryer) latestUpgradeKymaOperationValidate(orchestrationID string, ops []internal.UpgradeKymaOperation) ([]internal.UpgradeKymaOperation, []string, error) {
	var retryOps []internal.UpgradeKymaOperation
	var oldIDs []string

	for _, op := range ops {
		instanceID := op.InstanceID

		kymaOps, err := r.operations.ListUpgradeKymaOperationsByInstanceID(instanceID)
		if err != nil {
			// fail for listing operations of one instance, then http return and report fail
			r.log.Errorf("when getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "when getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		var errFound, newerExist bool
		num := len(kymaOps)

		for i := num - 1; i >= 0; i-- {
			if op.Operation.ID != kymaOps[i].Operation.ID {
				if num == 1 {
					errFound = true
					break
				}

				// not consider 'canceled' newer op as the conflict
				if kymaOps[i].State == commonOrchestration.Canceled {
					continue
				}

				oldIDs = append(oldIDs, op.Operation.ID)
				newerExist = true
			}

			break
		}

		if num == 0 || errFound {
			r.log.Errorf("when getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "when getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		if newerExist {
			continue
		}

		retryOps = append(retryOps, op)
	}

	return retryOps, oldIDs, nil
}

func (r *Retryer) latestUpgradeClusterOperationValidate(orchestrationID string, ops []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, []string, error) {
	var retryOps []internal.UpgradeClusterOperation
	var oldIDs []string

	for _, op := range ops {
		instanceID := op.InstanceID

		clusterOps, err := r.operations.ListUpgradeClusterOperationsByInstanceID(instanceID)
		if err != nil {
			// fail for listing operations of one instance, then http return and report fail
			r.log.Errorf("when getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "when getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		var errFound, newerExist bool
		num := len(clusterOps)

		for i := num - 1; i >= 0; i-- {
			if op.Operation.ID != clusterOps[i].Operation.ID {
				if num == 1 {
					errFound = true
					break
				}

				// not consider 'canceled' newer op as the conflict
				if clusterOps[i].State == commonOrchestration.Canceled {
					continue
				}

				oldIDs = append(oldIDs, op.Operation.ID)
				newerExist = true
			}

			break
		}

		if num == 0 || errFound {
			r.log.Errorf("when getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "when getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		if newerExist {
			continue
		}

		retryOps = append(retryOps, op)
	}

	return retryOps, oldIDs, nil
}

// filter out the operation which doesn't belong to the given orchestration
func (r *Retryer) kymaOrchestrationOperationsFilter(opsByOrch []internal.UpgradeKymaOperation, opsIDs []string) ([]internal.UpgradeKymaOperation, []string) {
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

func (r *Retryer) clusterOrchestrationOperationsFilter(opsByOrch []internal.UpgradeClusterOperation, opsIDs []string) ([]internal.UpgradeClusterOperation, []string) {
	if len(opsIDs) <= 0 {
		return opsByOrch, nil
	}

	var retOps []internal.UpgradeClusterOperation
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
