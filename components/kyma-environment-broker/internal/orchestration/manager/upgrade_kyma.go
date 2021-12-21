package manager

import (
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type upgradeKymaFactory struct {
	operationStorage          storage.Operations
	smcf                      internal.SMClientFactory
	defaultKymaVersion        string
	defaultKymaPreviewVersion string
}

func NewUpgradeKymaManager(orchestrationStorage storage.Orchestrations, operationStorage storage.Operations, instanceStorage storage.Instances,
	kymaUpgradeExecutor orchestration.OperationExecutor, resolver orchestration.RuntimeResolver, pollingInterval time.Duration,
	smcf internal.SMClientFactory, log logrus.FieldLogger, cli client.Client, cfg *internalOrchestration.Config) process.Executor {
	return &orchestrationManager{
		orchestrationStorage: orchestrationStorage,
		operationStorage:     operationStorage,
		instanceStorage:      instanceStorage,
		resolver:             resolver,
		factory: &upgradeKymaFactory{
			operationStorage:          operationStorage,
			smcf:                      smcf,
			defaultKymaVersion:        cfg.KymaVersion,
			defaultKymaPreviewVersion: cfg.KymaPreviewVersion,
		},
		executor:          kymaUpgradeExecutor,
		pollingInterval:   pollingInterval,
		log:               log,
		k8sClient:         cli,
		configNamespace:   cfg.Namespace,
		configName:        cfg.Name,
		kymaVersion:       cfg.KymaVersion,
		kubernetesVersion: cfg.KubernetesVersion,
	}
}

func (u *upgradeKymaFactory) NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance) (orchestration.RuntimeOperation, error) {
	id := uuid.New().String()
	details, err := i.GetInstanceDetails()
	if err != nil {
		return orchestration.RuntimeOperation{}, err
	}
	op := internal.UpgradeKymaOperation{
		Operation: internal.Operation{
			ID:                     id,
			Version:                0,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   internal.OperationTypeUpgradeKyma,
			InstanceID:             r.InstanceID,
			State:                  orchestration.Pending,
			Description:            "Operation created",
			OrchestrationID:        o.OrchestrationID,
			ProvisioningParameters: i.Parameters,
			InstanceDetails:        details,
		},
		RuntimeOperation: orchestration.RuntimeOperation{
			ID:      id,
			Runtime: r,
			DryRun:  o.Parameters.DryRun,
		},
		SMClientFactory: u.smcf,
	}
	if o.Parameters.Kyma.Version != "" {
		var majorVer int
		var err error

		if broker.IsPreviewPlan(i.ServicePlanID) {
			majorVer, err = determineMajorVersion(o.Parameters.Kyma.Version, u.defaultKymaPreviewVersion)
			if err != nil {
				return orchestration.RuntimeOperation{}, errors.Wrap(err, "while determining Kyma's major version")
			}
		} else {
			majorVer, err = determineMajorVersion(o.Parameters.Kyma.Version, u.defaultKymaVersion)
			if err != nil {
				return orchestration.RuntimeOperation{}, errors.Wrap(err, "while determining Kyma's major version")
			}
		}

		op.RuntimeVersion = *internal.NewRuntimeVersionFromParameters(o.Parameters.Kyma.Version, majorVer)
	}

	err = u.operationStorage.InsertUpgradeKymaOperation(op)
	return op.RuntimeOperation, err
}

func determineMajorVersion(version string, defaultVersion string) (int, error) {
	if isCustomVersion(version) {
		return extractMajorVersionNumberFromVersionString(defaultVersion)
	}
	return extractMajorVersionNumberFromVersionString(version)
}

func isCustomVersion(version string) bool {
	return strings.HasPrefix(version, "PR-") || strings.HasPrefix(version, "main-")
}

func extractMajorVersionNumberFromVersionString(version string) (int, error) {
	splitVer := strings.Split(version, ".")
	majorVerNum, err := strconv.Atoi(splitVer[0])
	if err != nil {
		return 0, errors.New("cannot convert major version to int")
	}
	return majorVerNum, nil
}

func (u *upgradeKymaFactory) ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error) {
	ops, _, _, err := u.operationStorage.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.InProgress, orchestration.Pending}})
	if err != nil {
		return nil, err
	}

	pending := make([]orchestration.RuntimeOperation, 0)
	inProgress := make([]orchestration.RuntimeOperation, 0)
	for _, op := range ops {
		if op.State == orchestration.Pending {
			pending = append(pending, op.RuntimeOperation)
		}
		if op.State == orchestration.InProgress {
			inProgress = append(inProgress, op.RuntimeOperation)
		}
	}

	return append(inProgress, pending...), nil
}

func (u *upgradeKymaFactory) CancelOperations(orchestrationID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return errors.Wrap(err, "while listing upgrade operations")
	}
	for _, op := range ops {
		op.State = orchestration.Canceled
		op.Description = "Operation was canceled"
		_, err := u.operationStorage.UpdateUpgradeKymaOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating upgrade kyma operation")
		}
	}

	return nil
}
