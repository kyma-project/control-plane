package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type orchestrations struct {
	dbsession.Factory
}

func NewOrchestrations(sess dbsession.Factory) *orchestrations {
	return &orchestrations{
		Factory: sess,
	}
}

func (s *orchestrations) Insert(orchestration orchestration.Orchestration) error {
	_, err := s.GetByID(orchestration.OrchestrationID)
	if err == nil {
		return dberr.AlreadyExists("orchestration with id %s already exist", orchestration.OrchestrationID)
	}

	dto, err := dbmodel.NewOrchestrationDTO(orchestration)
	if err != nil {
		return errors.Wrapf(err, "while converting Orchestration to DTO")
	}

	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertOrchestration(dto)
		if err != nil {
			log.Warn(errors.Wrapf(err, "while saving orchestration ID %s", orchestration.OrchestrationID).Error())
			return false, nil
		}
		return true, nil
	})
}

func (s *orchestrations) GetByID(orchestrationID string) (*orchestration.Orchestration, error) {
	sess := s.NewReadSession()
	orchestration := orchestration.Orchestration{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		var dto dbmodel.OrchestrationDTO
		dto, lastErr = sess.GetOrchestrationByID(orchestrationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestration with id %s not exist", orchestrationID)
			}
			log.Warn(errors.Wrapf(lastErr, "while getting orchestration by ID %s", orchestrationID).Error())
			return false, nil
		}
		orchestration, lastErr = dto.ToOrchestration()
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return &orchestration, nil
}

func (s *orchestrations) List(filter dbmodel.OrchestrationFilter) ([]orchestration.Orchestration, int, int, error) {
	sess := s.NewReadSession()
	var (
		orchestrations    = make([]orchestration.Orchestration, 0)
		lastErr           error
		count, totalCount int
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		var dtos []dbmodel.OrchestrationDTO
		dtos, count, totalCount, lastErr = sess.ListOrchestrations(filter)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestrations not exist")
			}
			log.Warn(errors.Wrapf(lastErr, "while getting orchestration").Error())
			return false, nil
		}
		for _, dto := range dtos {
			var o orchestration.Orchestration
			o, lastErr = dto.ToOrchestration()
			if lastErr != nil {
				return false, lastErr
			}
			orchestrations = append(orchestrations, o)
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, lastErr
	}
	return orchestrations, count, totalCount, nil
}

func (s *orchestrations) Update(orchestration orchestration.Orchestration) error {
	dto, err := dbmodel.NewOrchestrationDTO(orchestration)
	if err != nil {
		return errors.Wrapf(err, "while converting Orchestration to DTO")
	}

	sess := s.NewWriteSession()
	var lastErr dberr.Error
	err = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = sess.UpdateOrchestration(dto)
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

func (s *orchestrations) ListByState(state string) ([]orchestration.Orchestration, error) {
	sess := s.NewReadSession()
	var (
		lastErr error
		result  []orchestration.Orchestration
		filter  = dbmodel.OrchestrationFilter{
			States: []string{state},
		}
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		var dtos []dbmodel.OrchestrationDTO
		dtos, _, _, lastErr = sess.ListOrchestrations(filter)
		if lastErr != nil {
			log.Warnf("while listing %s orchestrations: %v", state, lastErr)
			return false, nil
		}
		for _, dto := range dtos {
			var o orchestration.Orchestration
			o, lastErr = dto.ToOrchestration()
			if lastErr != nil {
				return false, lastErr
			}
			result = append(result, o)
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return result, nil
}
