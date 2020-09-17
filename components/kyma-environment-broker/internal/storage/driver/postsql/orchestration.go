package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type orchestration struct {
	dbsession.Factory
}

func NewOrchestrations(sess dbsession.Factory) *orchestration {
	return &orchestration{
		Factory: sess,
	}
}

func (s *orchestration) Insert(orchestration internal.Orchestration) error {
	_, err := s.GetByID(orchestration.OrchestrationID)
	if err == nil {
		return dberr.AlreadyExists("orchestration with id %s already exist", orchestration.OrchestrationID)
	}

	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertOrchestration(orchestration)
		if err != nil {
			log.Warn(errors.Wrapf(err, "while saving orchestration ID %s", orchestration.OrchestrationID).Error())
			return false, nil
		}
		return true, nil
	})
}
func (s *orchestration) GetByID(orchestrationID string) (*internal.Orchestration, error) {
	sess := s.NewReadSession()
	orchestration := internal.Orchestration{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		orchestration, lastErr = sess.GetOrchestrationByID(orchestrationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestration with id %s not exist", orchestrationID)
			}
			log.Warn(errors.Wrapf(lastErr, "while getting orchestration by ID %s", orchestrationID).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return &orchestration, nil
}

func (s *orchestration) ListAll() ([]internal.Orchestration, error) {
	sess := s.NewReadSession()
	orchestrations := make([]internal.Orchestration, 0)
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		orchestrations, lastErr = sess.ListOrchestrations()
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestrations not exist")
			}
			log.Warn(errors.Wrapf(lastErr, "while getting orchestration").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return orchestrations, nil
}

func (s *orchestration) Update(orchestration internal.Orchestration) error {
	sess := s.NewWriteSession()
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = sess.UpdateOrchestration(orchestration)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestration with id %s not exist", orchestration.OrchestrationID)
			}
			log.Warn(errors.Wrapf(lastErr, "while updating orchestration ID %s", orchestration.OrchestrationID).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return lastErr
	}
	return nil
}

func (s *orchestration) ListByState(state string) ([]internal.Orchestration, error) {
	sess := s.NewReadSession()
	var lastErr dberr.Error
	var result []internal.Orchestration
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		result, lastErr = sess.ListOrchestrationsByState(state)
		if lastErr != nil {
			log.Warnf("while listing %s orchestrations: %v", state, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return result, nil
}
