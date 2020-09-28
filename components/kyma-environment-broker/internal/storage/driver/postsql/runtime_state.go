package postsql

import (
	"encoding/json"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type runtimeState struct {
	dbsession.Factory

	cipher Cipher
}

func NewRuntimeStates(sess dbsession.Factory, cipher Cipher) *runtimeState {
	return &runtimeState{
		Factory: sess,
		cipher:  cipher,
	}
}

func (s *runtimeState) Insert(runtimeState internal.RuntimeState) error {
	state, err := runtimeStateToDB(runtimeState)
	if err != nil {
		return err
	}
	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertRuntimeState(state)
		if err != nil {
			log.Warnf("while saving orchestration ID %s: %v", runtimeState.ID, err)
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
			log.Warnf("while getting RuntimeState: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	result, err := toRuntimeStates(states)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func runtimeStateToDB(op internal.RuntimeState) (dbmodel.RuntimeStateDTO, error) {
	kymaCfg, err := json.Marshal(op.KymaConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, errors.Wrap(err, "while encoding kyma config")
	}
	clusterCfg, err := json.Marshal(op.ClusterConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, errors.Wrap(err, "while encoding cluster config")
	}

	return dbmodel.RuntimeStateDTO{
		ID:            op.ID,
		CreatedAt:     op.CreatedAt,
		RuntimeID:     op.RuntimeID,
		OperationID:   op.OperationID,
		KymaConfig:    string(kymaCfg),
		ClusterConfig: string(clusterCfg),
		KymaVersion:   op.KymaConfig.Version,
		K8SVersion:    op.ClusterConfig.KubernetesVersion,
	}, nil
}

func toRuntimeState(dto *dbmodel.RuntimeStateDTO) (internal.RuntimeState, error) {
	var (
		kymaCfg    gqlschema.KymaConfigInput
		clusterCfg gqlschema.GardenerConfigInput
	)
	if dto.KymaConfig != "" {
		if err := json.Unmarshal([]byte(dto.KymaConfig), &kymaCfg); err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while unmarshall kyma config")
		}
	}
	if dto.ClusterConfig != "" {
		if err := json.Unmarshal([]byte(dto.ClusterConfig), &clusterCfg); err != nil {
			return internal.RuntimeState{}, errors.Wrap(err, "while unmarshall cluster config")
		}
	}
	return internal.RuntimeState{
		ID:            dto.ID,
		CreatedAt:     dto.CreatedAt,
		RuntimeID:     dto.RuntimeID,
		OperationID:   dto.OperationID,
		KymaConfig:    kymaCfg,
		ClusterConfig: clusterCfg,
	}, nil
}

func toRuntimeStates(states []dbmodel.RuntimeStateDTO) ([]internal.RuntimeState, error) {
	result := make([]internal.RuntimeState, 0)

	for _, state := range states {
		r, err := toRuntimeState(&state)
		if err != nil {
			return nil, errors.Wrap(err, "while converting runtime states")
		}
		result = append(result, r)
	}

	return result, nil
}
