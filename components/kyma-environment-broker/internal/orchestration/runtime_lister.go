package orchestration

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	runtimeInt "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RuntimeLister struct {
	instancesDb  storage.Instances
	operationsDb storage.Operations
	converter    runtimeInt.Converter
	log          logrus.FieldLogger
}

func NewRuntimeLister(instancesDb storage.Instances, operationsDb storage.Operations, converter runtimeInt.Converter, log logrus.FieldLogger) *RuntimeLister {
	return &RuntimeLister{
		instancesDb:  instancesDb,
		operationsDb: operationsDb,
		converter:    converter,
		log:          log,
	}
}

func (rl RuntimeLister) ListAllRuntimes() ([]runtime.RuntimeDTO, error) {
	instances, _, _, err := rl.instancesDb.List(dbmodel.InstanceFilter{})
	if err != nil {
		return nil, errors.Wrap(err, "while listing instances from DB")
	}

	runtimes := make([]runtime.RuntimeDTO, 0, len(instances))
	for _, inst := range instances {
		dto, err := rl.converter.NewDTO(inst)
		if err != nil {
			rl.log.Errorf("cannot convert instance to DTO: %s", err.Error())
			continue
		}

		pOpr, err := rl.operationsDb.GetProvisioningOperationByInstanceID(inst.InstanceID)
		if err != nil {
			rl.log.Errorf("while getting provision operation for instance %s: %s", inst.InstanceID, err.Error())
			continue
		}
		rl.converter.ApplyProvisioningOperation(&dto, pOpr)

		dOpr, err := rl.operationsDb.GetDeprovisioningOperationByInstanceID(inst.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			rl.log.Errorf("while getting deprovision operation for instance %s: %s", inst.InstanceID, err.Error())
			continue
		}
		rl.converter.ApplyDeprovisioningOperation(&dto, dOpr)

		runtimes = append(runtimes, dto)
	}

	return runtimes, nil
}
