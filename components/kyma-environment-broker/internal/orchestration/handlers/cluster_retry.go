package handlers

import (
	"time"

	commonOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type clusterRetryer Retryer

func NewClusterRetryer(orchestrations storage.Orchestrations, operations storage.Operations, q *process.Queue, logger logrus.FieldLogger) *clusterRetryer {
	return &clusterRetryer{
		orchestrations: orchestrations,
		operations:     operations,
		queue:          q,
		log:            logger,
	}
}

func (r *clusterRetryer) orchestrationRetry(o *internal.Orchestration, opsByOrch []internal.UpgradeClusterOperation, operationIDs []string) (commonOrchestration.RetryResponse, error) {
	var err error
	resp := commonOrchestration.RetryResponse{OrchestrationID: o.OrchestrationID}

	ops, invalidIDs := r.orchestrationOperationsFilter(opsByOrch, operationIDs)
	resp.InvalidOperations = invalidIDs
	if len(ops) == 0 {
		zeroValidOperationInfo(&resp, r.log)
		return resp, nil
	}

	// as failed orchestration has finished before
	// only retry the latest failed cluster upgrade operation for the same instance
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

	err = r.OperationsStateUpdate(ops)
	if err != nil {
		return resp, err
	}

	// get orchestration state again in case in progress changed to failed, need to put in queue
	lastState, err := orchestrationStateUpdate(r.orchestrations, o.OrchestrationID, r.log)
	if err != nil {
		return resp, err
	}

	if lastState == commonOrchestration.Failed {
		r.queue.Add(o.OrchestrationID)
	}

	return resp, nil
}

func (r *clusterRetryer) orchestrationOperationsFilter(opsByOrch []internal.UpgradeClusterOperation, opsIDs []string) ([]internal.UpgradeClusterOperation, []string) {
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

func (r *clusterRetryer) latestOperationValidate(orchestrationID string, ops []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, []string, error) {
	var retryOps []internal.UpgradeClusterOperation
	var oldIDs []string

	for _, op := range ops {
		instanceID := op.InstanceID

		clusterOps, err := r.operations.ListUpgradeClusterOperationsByInstanceID(instanceID)
		if err != nil {
			// fail for listing operations of one instance, then http return and report fail
			r.log.Errorf("while getting operations by instanceID %s: %v", instanceID, err)
			err = errors.Wrapf(err, "while getting operations by instanceID %s", instanceID)
			return nil, nil, err
		}

		var errFound, newerExist bool
		num := len(clusterOps)

		for i := 0; i < num; i++ {
			if op.CreatedAt.Before(clusterOps[i].CreatedAt) {
				if num == 1 {
					errFound = true
					break
				}

				// 'canceled' or 'canceling' newer op is not a newer op
				if clusterOps[i].State == commonOrchestration.Canceled || clusterOps[i].State == commonOrchestration.Canceling {
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

func (r *clusterRetryer) OperationsStateUpdate(ops []internal.UpgradeClusterOperation) error {
	for _, op := range ops {
		op.State = commonOrchestration.Retrying
		op.UpdatedAt = time.Now()
		op.Description = "queued for retrying"

		_, err := r.operations.UpdateUpgradeClusterOperation(op)
		if err != nil {
			// one update fail then http return
			r.log.Errorf("Cannot update operation %s in storage: %s", op.Operation.ID, err)
			return errors.Wrapf(err, "while updating orchestration %s", op.OrchestrationID)
		}
	}

	return nil
}
