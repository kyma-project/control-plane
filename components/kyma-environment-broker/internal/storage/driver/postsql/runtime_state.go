package postsql

import (
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	contract "github.com/kyma-incubator/reconciler/pkg/keb"
)

type runtimeState struct {
	postsql.Factory

	cipher Cipher
}

func NewRuntimeStates(sess postsql.Factory, cipher Cipher) *runtimeState {
	return &runtimeState{
		Factory: sess,
		cipher:  cipher,
	}
}

func (s *runtimeState) Insert(runtimeState internal.RuntimeState) error {
	state, err := s.runtimeStateToDB(runtimeState)
	if err != nil {
		return err
	}
	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertRuntimeState(state)
		if err != nil {
			log.Errorf("while saving runtime state ID %s: %v", runtimeState.ID, err)
			return false, nil
		}
		return true, nil
	})
}

func (s *runtimeState) ListByRuntimeID(runtimeID string) ([]internal.RuntimeState, error) {
	sess := s.NewReadSession()
	states := make([]dbmodel.RuntimeStateDTO, 0)
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		states, lastErr = sess.ListRuntimeStateByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeStates not found")
			}
			log.Errorf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	result, err := s.toRuntimeStates(states)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *runtimeState) GetByOperationID(operationID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	state := dbmodel.RuntimeStateDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetRuntimeStateByOperationID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for operation %s not found", operationID)
			}
			log.Errorf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, errors.Wrap(err, "while converting runtime states")
	}

	return result, nil
}

func (s *runtimeState) GetLatestByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	var state dbmodel.RuntimeStateDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetLatestRuntimeStateByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for runtime %s not found", runtimeID)
			}
			log.Errorf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, errors.Wrap(err, "while converting runtime state")
	}

	return result, nil
}

func (s *runtimeState) GetLatestWithReconcilerInputByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	var state dbmodel.RuntimeStateDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetLatestRuntimeStateWithReconcilerInputByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for runtime %s not found", runtimeID)
			}
			log.Errorf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, errors.Wrap(err, "while converting runtime state")
	}

	return result, nil
}

func (s *runtimeState) GetLatestWithKymaVersionByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	var states []dbmodel.RuntimeStateDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		states, lastErr = sess.GetLatestRuntimeStatesByRuntimeID(runtimeID, 100)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for runtime %s not found", runtimeID)
			}
			log.Errorf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	for _, state := range states {
		result, err := s.toRuntimeState(&state)
		if err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while converting runtime state")
		}
		if result.ClusterSetup != nil && result.ClusterSetup.KymaConfig.Version != "" {
			return result, nil
		}
		if result.KymaConfig.Version != "" {
			return result, nil
		}
	}

	return internal.RuntimeState{}, fmt.Errorf("failed to find RuntimeState with kyma version for runtime %s", runtimeID)
}

func (s *runtimeState) runtimeStateToDB(state internal.RuntimeState) (dbmodel.RuntimeStateDTO, error) {
	kymaCfg, err := json.Marshal(state.KymaConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, errors.Wrap(err, "while encoding kyma config")
	}
	clusterCfg, err := json.Marshal(state.ClusterConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, errors.Wrap(err, "while encoding cluster config")
	}
	clusterSetup, err := s.provideClusterSetup(state.ClusterSetup)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, err
	}

	encKymaCfg, err := s.cipher.Encrypt(kymaCfg)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, errors.Wrap(err, "while encrypting kyma config")
	}

	return dbmodel.RuntimeStateDTO{
		ID:            state.ID,
		CreatedAt:     state.CreatedAt,
		RuntimeID:     state.RuntimeID,
		OperationID:   state.OperationID,
		KymaConfig:    string(encKymaCfg),
		ClusterConfig: string(clusterCfg),
		ClusterSetup:  string(clusterSetup),
		KymaVersion:   state.KymaConfig.Version,
		K8SVersion:    state.ClusterConfig.KubernetesVersion,
	}, nil
}

func (s *runtimeState) toRuntimeState(dto *dbmodel.RuntimeStateDTO) (internal.RuntimeState, error) {
	var (
		kymaCfg      gqlschema.KymaConfigInput
		clusterCfg   gqlschema.GardenerConfigInput
		clusterSetup *contract.Cluster
	)
	if dto.KymaConfig != "" {
		cfg, err := s.cipher.Decrypt([]byte(dto.KymaConfig))
		if err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while decrypting kyma config")
		}
		if err := json.Unmarshal(cfg, &kymaCfg); err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while unmarshall kyma config")
		}
	}
	if dto.ClusterConfig != "" {
		if err := json.Unmarshal([]byte(dto.ClusterConfig), &clusterCfg); err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while unmarshall cluster config")
		}
	}
	if dto.ClusterSetup != "" {
		setup, err := s.cipher.Decrypt([]byte(dto.ClusterSetup))
		if err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while decrypting cluster setup")
		}
		clusterSetup = &contract.Cluster{}
		if err := json.Unmarshal(setup, clusterSetup); err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while unmarshall cluster setup")
		}
	}
	return internal.RuntimeState{
		ID:            dto.ID,
		CreatedAt:     dto.CreatedAt,
		RuntimeID:     dto.RuntimeID,
		OperationID:   dto.OperationID,
		KymaConfig:    kymaCfg,
		ClusterConfig: clusterCfg,
		ClusterSetup:  clusterSetup,
	}, nil
}

func (s *runtimeState) toRuntimeStates(states []dbmodel.RuntimeStateDTO) ([]internal.RuntimeState, error) {
	result := make([]internal.RuntimeState, 0)

	for _, state := range states {
		r, err := s.toRuntimeState(&state)
		if err != nil {
			return nil, errors.Wrap(err, "while converting runtime states")
		}
		result = append(result, r)
	}

	return result, nil
}

func (s *runtimeState) provideClusterSetup(clusterSetup *contract.Cluster) ([]byte, error) {
	marshalledClusterSetup, err := s.marshalClusterSetup(clusterSetup)
	if err != nil {
		return nil, errors.Wrap(err, "while encoding reconciler input")
	}
	encryptedClusterSetup, err := s.encryptClusterSetup(marshalledClusterSetup)
	if err != nil {
		return nil, errors.Wrap(err, "while encrypting reconciler input")
	}
	return encryptedClusterSetup, nil
}

func (s *runtimeState) marshalClusterSetup(clusterSetup *contract.Cluster) ([]byte, error) {
	var (
		result []byte
		err    error
	)
	if clusterSetup != nil {
		result, err = json.Marshal(clusterSetup)
		if err != nil {
			return nil, err
		}
	} else {
		result = make([]byte, 0, 0)
	}
	return result, nil
}

func (s *runtimeState) encryptClusterSetup(marshalledClusterSetup []byte) ([]byte, error) {
	if string(marshalledClusterSetup) == "" {
		return marshalledClusterSetup, nil
	}
	return s.cipher.Encrypt(marshalledClusterSetup)
}
