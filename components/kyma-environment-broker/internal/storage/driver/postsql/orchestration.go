package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type orchestrations struct {
	postsql.Factory
}

func NewOrchestrations(sess postsql.Factory) *orchestrations {
	return &orchestrations{
		Factory: sess,
	}
}

func (s *orchestrations) Insert(orchestration internal.Orchestration) error {
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
			log.Errorf("while saving orchestration ID %s: %v", orchestration.OrchestrationID, err)
			return false, nil
		}
		return true, nil
	})
}

func (s *orchestrations) GetByID(orchestrationID string) (*internal.Orchestration, error) {
	sess := s.NewReadSession()
	orchestration := internal.Orchestration{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		var dto dbmodel.OrchestrationDTO
		dto, lastErr = sess.GetOrchestrationByID(orchestrationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Orchestration with id %s not exist", orchestrationID)
			}
			log.Errorf("while getting orchestration by ID %s: %v", orchestrationID, lastErr)
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

func (s *orchestrations) List(filter dbmodel.OrchestrationFilter) ([]internal.Orchestration, int, int, error) {
	sess := s.NewReadSession()
	var (
		orchestrations    = make([]internal.Orchestration, 0)
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
			log.Errorf("while getting orchestration: %v", lastErr)
			return false, nil
		}
		for _, dto := range dtos {
			var o internal.Orchestration
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

func (s *orchestrations) Update(orchestration internal.Orchestration) error {
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
			log.Errorf("while updating orchestration ID %s: %v", orchestration.OrchestrationID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return lastErr
	}
	return nil
}

func (s *orchestrations) ListByState(state string) ([]internal.Orchestration, error) {
	sess := s.NewReadSession()
	var (
		lastErr error
		result  []internal.Orchestration
		filter  = dbmodel.OrchestrationFilter{
			States: []string{state},
		}
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		var dtos []dbmodel.OrchestrationDTO
		dtos, _, _, lastErr = sess.ListOrchestrations(filter)
		if lastErr != nil {
			log.Errorf("while listing %s orchestrations: %v", state, lastErr)
			return false, nil
		}
		for _, dto := range dtos {
			var o internal.Orchestration
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
