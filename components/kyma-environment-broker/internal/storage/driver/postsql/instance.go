package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Instance struct {
	postsql.Factory
}

func NewInstance(sess postsql.Factory) *Instance {
	return &Instance{
		Factory: sess,
	}
}

func (s *Instance) FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error) {
	sess := s.NewReadSession()
	var (
		instances []internal.InstanceWithOperation
		lastErr   dberr.Error
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesJoinedWithOperation(prct...)
		if lastErr != nil {
			log.Errorf("while fetching all instances: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	return instances, nil
}

func (s *Instance) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error) {
	sess := s.NewReadSession()
	var instances []internal.Instance
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesForRuntimes(runtimeIdList)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instances with runtime IDs from list '%+q' not exist", runtimeIdList)
			}
			log.Errorf("while getting instances from runtime ID list '%+q': %v", runtimeIdList, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return instances, nil
}

func (s *Instance) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error) {
	sess := s.NewReadSession()
	var (
		instances []internal.Instance
		lastErr   dberr.Error
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesForSubAccounts(subAccountslist)
		if lastErr != nil {
			log.Errorf("while fetching instances by subaccount list: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	return instances, nil
}

func (s *Instance) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	sess := s.NewReadSession()
	var result int
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		count, err := sess.GetNumberOfInstancesForGlobalAccountID(globalAccountID)
		result = count
		return err == nil, nil
	})
	return result, err
}

// TODO: Wrap retries in single method WithRetries
func (s *Instance) GetByID(instanceID string) (*internal.Instance, error) {
	sess := s.NewReadSession()
	instance := internal.Instance{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instance, lastErr = sess.GetInstanceByID(instanceID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instanceID)
			}
			log.Errorf("while getting instance by ID %s: %v", instanceID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return &instance, nil
}

func (s *Instance) Insert(instance internal.Instance) error {
	_, err := s.GetByID(instance.InstanceID)
	if err == nil {
		return dberr.AlreadyExists("instance with id %s already exist", instance.InstanceID)
	}

	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertInstance(instance)
		if err != nil {
			log.Errorf("while saving instance ID %s: %v", instance.InstanceID, err)
			return false, nil
		}
		return true, nil
	})
}

func (s *Instance) Update(instance internal.Instance) (*internal.Instance, error) {
	sess := s.NewWriteSession()
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = sess.UpdateInstance(instance)

		switch {
		case dberr.IsNotFound(lastErr):
			_, lastErr = s.NewReadSession().GetInstanceByID(instance.InstanceID)
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instance.InstanceID)
			}
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", instance.InstanceID)
			return false, lastErr
		case lastErr != nil:
			log.Errorf("while updating instance ID %s: %v", instance.InstanceID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance.Version = instance.Version + 1
	return &instance, nil
}

func (s *Instance) Delete(instanceID string) error {
	sess := s.NewWriteSession()
	return sess.DeleteInstance(instanceID)
}

func (s *Instance) GetInstanceStats() (internal.InstanceStats, error) {
	entries, err := s.NewReadSession().GetInstanceStats()
	if err != nil {
		return internal.InstanceStats{}, err
	}

	result := internal.InstanceStats{
		PerGlobalAccountID: make(map[string]int),
	}
	for _, e := range entries {
		result.PerGlobalAccountID[e.GlobalAccountID] = e.Total
		result.TotalNumberOfInstances = result.TotalNumberOfInstances + e.Total
	}
	return result, nil
}

func (s *Instance) List(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	return s.NewReadSession().ListInstances(filter)
}
