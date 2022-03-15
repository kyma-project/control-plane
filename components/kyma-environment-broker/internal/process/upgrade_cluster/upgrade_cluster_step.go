package upgrade_cluster

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DryRunPrefix = "dry_run-"

type UpgradeClusterStep struct {
	operationManager    *process.UpgradeClusterOperationManager
	provisionerClient   provisioner.Client
	runtimeStateStorage storage.RuntimeStates
	timeSchedule        TimeSchedule
}

func NewUpgradeClusterStep(
	os storage.Operations,
	runtimeStorage storage.RuntimeStates,
	cli provisioner.Client,
	timeSchedule *TimeSchedule) *UpgradeClusterStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:                 5 * time.Second,
			StatusCheck:           time.Minute,
			UpgradeClusterTimeout: time.Hour,
		}
	}

	return &UpgradeClusterStep{
		operationManager:    process.NewUpgradeClusterOperationManager(os),
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
		timeSchedule:        *ts,
	}
}

func (s *UpgradeClusterStep) Name() string {
	return "Upgrade_Cluster"
}

func (s *UpgradeClusterStep) Run(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.timeSchedule.UpgradeClusterTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.timeSchedule.UpgradeClusterTimeout), nil, log)
	}

	lastRuntimeState, err := s.runtimeStateStorage.GetLatestByRuntimeID(operation.InstanceDetails.RuntimeID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 1*time.Minute, log)
	}

	input, err := s.createUpgradeShootInput(operation, &lastRuntimeState.ClusterConfig)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgradeShoot input", err, log)
	}

	if operation.DryRun {
		// runtimeID is set with prefix to indicate the fake runtime state
		err = s.runtimeStateStorage.Insert(
			internal.NewRuntimeState(fmt.Sprintf("%s%s", DryRunPrefix, operation.RuntimeOperation.RuntimeID), operation.Operation.ID, nil, gardenerUpgradeInputToConfigInput(input)),
		)
		if err != nil {
			return operation, 10 * time.Second, nil
		}
		return s.operationManager.OperationSucceeded(operation, "dry run succeeded", log)
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		// trigger upgradeRuntime mutation
		provisionerResponse, err = s.provisionerClient.UpgradeShoot(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeOperation.RuntimeID, input)
		if err != nil {
			log.Errorf("call to provisioner failed: %s", err)
			return operation, s.timeSchedule.Retry, nil
		}

		repeat := time.Duration(0)
		operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(op *internal.UpgradeClusterOperation) {
			op.ProvisionerOperationID = *provisionerResponse.ID
			op.Description = "cluster upgrade in progress"
		}, log)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner")
			return operation, s.timeSchedule.Retry, nil
		}
	}

	if provisionerResponse.RuntimeID == nil {
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
		if err != nil {
			log.Errorf("call to provisioner about operation status failed: %s", err)
			return operation, s.timeSchedule.Retry, nil
		}
	}
	if provisionerResponse.RuntimeID == nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log = log.WithField("runtimeID", *provisionerResponse.RuntimeID)
	log.Infof("call to provisioner succeeded, got operation ID %q", *provisionerResponse.ID)

	rs := internal.NewRuntimeState(*provisionerResponse.RuntimeID, operation.Operation.ID, nil, gardenerUpgradeInputToConfigInput(input))
	err = s.runtimeStateStorage.Insert(rs)
	if err != nil {
		log.Errorf("cannot insert runtimeState: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("cluster upgrade process initiated successfully")

	// return repeat mode to start the initialization step which will now check the runtime status
	return operation, s.timeSchedule.Retry, nil

}

func (s *UpgradeClusterStep) createUpgradeShootInput(operation internal.UpgradeClusterOperation, lastClusterConfig *gqlschema.GardenerConfigInput) (gqlschema.UpgradeShootInput, error) {
	operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	if lastClusterConfig.OidcConfig != nil {
		operation.InputCreator.SetOIDCLastValues(*lastClusterConfig.OidcConfig)
	}
	input, err := operation.InputCreator.CreateUpgradeShootInput()
	if err != nil {
		return input, errors.Wrap(err, "while building upgradeShootInput for provisioner")
	}

	return input, nil
}

func gardenerUpgradeInputToConfigInput(input gqlschema.UpgradeShootInput) *gqlschema.GardenerConfigInput {
	result := &gqlschema.GardenerConfigInput{
		MachineImage:                        input.GardenerConfig.MachineImage,
		MachineImageVersion:                 input.GardenerConfig.MachineImageVersion,
		DiskType:                            input.GardenerConfig.DiskType,
		VolumeSizeGb:                        input.GardenerConfig.VolumeSizeGb,
		Purpose:                             input.GardenerConfig.Purpose,
		OidcConfig:                          input.GardenerConfig.OidcConfig,
		EnableKubernetesVersionAutoUpdate:   input.GardenerConfig.EnableKubernetesVersionAutoUpdate,
		EnableMachineImageVersionAutoUpdate: input.GardenerConfig.EnableMachineImageVersionAutoUpdate,
	}
	if input.GardenerConfig.KubernetesVersion != nil {
		result.KubernetesVersion = *input.GardenerConfig.KubernetesVersion
	}
	if input.GardenerConfig.MachineType != nil {
		result.MachineType = *input.GardenerConfig.MachineType
	}
	if input.GardenerConfig.AutoScalerMin != nil {
		result.AutoScalerMin = *input.GardenerConfig.AutoScalerMin
	}
	if input.GardenerConfig.AutoScalerMax != nil {
		result.AutoScalerMax = *input.GardenerConfig.AutoScalerMax
	}
	if input.GardenerConfig.MaxSurge != nil {
		result.MaxSurge = *input.GardenerConfig.MaxSurge
	}
	if input.GardenerConfig.MaxUnavailable != nil {
		result.MaxUnavailable = *input.GardenerConfig.MaxUnavailable
	}

	return result
}
