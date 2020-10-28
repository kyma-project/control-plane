package orchestration

import (
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	brokerapi "github.com/pivotal-cf/brokerapi/v7/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerclient "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

type instanceOperationStatus struct {
	internal.Instance
	provisionState   brokerapi.LastOperationState
	deprovisionState brokerapi.LastOperationState
}

// InstanceLister is the interface to get InstanceWithOperation objects from KEB storage
//go:generate mockery -name=InstanceLister -output=automock -outpkg=automock -case=underscore
type InstanceLister interface {
	FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error)
}

// GardenerRuntimeResolver is the default resolver which implements the RuntimeResolver interface.
// This resolver uses the Shoot resources on the Gardener cluster to resolve the runtime targets.
//
// Naive implementation, listing all the shoots and perfom filtering on the result.
// The logic could be optimized with k8s client cache using shoot lister / indexer.
// The implementation is thread safe, i.e. it is safe to call Resolve() from multiple threads concurrently.
type GardenerRuntimeResolver struct {
	gardenerClient     gardenerclient.CoreV1beta1Interface
	gardenerNamespace  string
	instanceLister     InstanceLister
	instanceOperations map[string]*instanceOperationStatus
	instanceMutex      sync.RWMutex
	logger             logrus.FieldLogger
}

const (
	globalAccountLabel      = "account"
	subAccountLabel         = "subaccount"
	runtimeIDAnnotation     = "kcp.provisioner.kyma-project.io/runtime-id"
	maintenanceWindowFormat = "150405-0700"
)

// NewGardenerRuntimeResolver constructs a GardenerRuntimeResolver with the mandatory input parameters.
func NewGardenerRuntimeResolver(gardenerClient gardenerclient.CoreV1beta1Interface, gardenerNamespace string, lister InstanceLister, logger logrus.FieldLogger) *GardenerRuntimeResolver {
	return &GardenerRuntimeResolver{
		gardenerClient:     gardenerClient,
		gardenerNamespace:  gardenerNamespace,
		instanceLister:     lister,
		instanceOperations: map[string]*instanceOperationStatus{},
		logger:             logger.WithField("orchestration", "resolver"),
	}
}

// Resolve given an input slice of target specs to include and exclude, returns back a list of unique Runtime objects
func (resolver *GardenerRuntimeResolver) Resolve(targets orchestration.TargetSpec) ([]internal.Runtime, error) {
	runtimeIncluded := map[string]bool{}
	runtimeExcluded := map[string]bool{}
	runtimes := []internal.Runtime{}
	shoots, err := resolver.getAllShoots()
	if err != nil {
		return nil, errors.Wrapf(err, "while listing gardener shoots in namespace %s", resolver.gardenerNamespace)
	}
	err = resolver.syncInstanceOperations()
	if err != nil {
		return nil, errors.Wrap(err, "while listing instances and operations from DB")
	}

	// Assemble IDs of runtimes to exclude
	for _, rt := range targets.Exclude {
		runtimesToExclude, err := resolver.resolveRuntimeTarget(rt, shoots)
		if err != nil {
			return nil, err
		}
		for _, r := range runtimesToExclude {
			runtimeExcluded[r.RuntimeID] = true
		}
	}

	// Include runtimes which are not excluded
	for _, rt := range targets.Include {
		runtimesToAdd, err := resolver.resolveRuntimeTarget(rt, shoots)
		if err != nil {
			return nil, err
		}
		for _, r := range runtimesToAdd {
			if !runtimeExcluded[r.RuntimeID] && !runtimeIncluded[r.RuntimeID] {
				runtimeIncluded[r.RuntimeID] = true
				runtimes = append(runtimes, r)
			}
		}
	}

	return runtimes, nil
}

func (resolver *GardenerRuntimeResolver) getAllShoots() ([]gardenerapi.Shoot, error) {
	shootList, err := resolver.gardenerClient.Shoots(resolver.gardenerNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return shootList.Items, nil
}

func (resolver *GardenerRuntimeResolver) syncInstanceOperations() error {
	instances, err := resolver.instanceLister.FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
	if err != nil {
		return err
	}
	resolver.instanceMutex.Lock()
	defer resolver.instanceMutex.Unlock()

	for _, inst := range instances {
		runtimeOpStat := resolver.instanceOperations[inst.RuntimeID]
		if runtimeOpStat == nil {
			runtimeOpStat = &instanceOperationStatus{
				inst.Instance,
				"",
				"",
			}
			resolver.instanceOperations[inst.RuntimeID] = runtimeOpStat
		}
		switch dbmodel.OperationType(inst.Type.String) {
		case dbmodel.OperationTypeProvision:
			runtimeOpStat.provisionState = brokerapi.LastOperationState(inst.State.String)
		case dbmodel.OperationTypeDeprovision:
			runtimeOpStat.deprovisionState = brokerapi.LastOperationState(inst.State.String)
		}
	}

	return nil
}

func (resolver *GardenerRuntimeResolver) getInstanceOperationStatus(runtimeID string) *instanceOperationStatus {
	resolver.instanceMutex.RLock()
	defer resolver.instanceMutex.RUnlock()
	return resolver.instanceOperations[runtimeID]
}

func (resolver *GardenerRuntimeResolver) resolveRuntimeTarget(rt orchestration.RuntimeTarget, shoots []gardenerapi.Shoot) ([]internal.Runtime, error) {
	runtimes := []internal.Runtime{}

	// Iterate over all shoots. Evaluate target specs. If multiple are specified, all must match for a given shoot.
	for _, shoot := range shoots {
		// Skip runtimes for which
		//  - there is no succeeded instance provision operation in DB
		//  - deprovision operation exists in DB
		runtimeID := shoot.Annotations[runtimeIDAnnotation]
		if runtimeID == "" {
			resolver.logger.Errorf("Failed to get runtimeID from %s annotation for Shoot %s", runtimeIDAnnotation, shoot.Name)
			continue
		}
		instanceOpStatus := resolver.getInstanceOperationStatus(runtimeID)
		if instanceOpStatus == nil {
			resolver.logger.Errorf("Couldn't find InstanceOperationStatus for runtimeID %s", runtimeID)
			continue
		}
		if instanceOpStatus.provisionState != brokerapi.Succeeded || instanceOpStatus.deprovisionState != "" {
			resolver.logger.Infof("Skipping Shoot %s (runtimeID: %s, instanceID %s) due to provisioning/deprovisioning state: %s/%s", shoot.Name, runtimeID, instanceOpStatus.InstanceID, instanceOpStatus.provisionState, instanceOpStatus.deprovisionState)
			continue
		}
		maintenanceWindowBegin, err := time.Parse(maintenanceWindowFormat, shoot.Spec.Maintenance.TimeWindow.Begin)
		if err != nil {
			resolver.logger.Errorf("Failed to parse maintenanceWindowBegin value %s of shoot %s ", shoot.Spec.Maintenance.TimeWindow.Begin, shoot.Name)
			continue
		}
		maintenanceWindowEnd, err := time.Parse(maintenanceWindowFormat, shoot.Spec.Maintenance.TimeWindow.End)
		if err != nil {
			resolver.logger.Errorf("Failed to parse maintenanceWindowEnd value %s of shoot %s ", shoot.Spec.Maintenance.TimeWindow.End, shoot.Name)
			continue
		}

		// Match exact shoot by runtimeID
		if rt.RuntimeID != "" {
			if rt.RuntimeID == runtimeID {
				runtimes = append(runtimes, resolver.runtimeFromOperationStatus(instanceOpStatus, shoot.Name, maintenanceWindowBegin, maintenanceWindowEnd))
			}
			continue
		}

		// Perform match against a specific PlanName
		if rt.PlanName != "" {
			if rt.PlanName != instanceOpStatus.ServicePlanName {
				continue
			}
		}

		// Perform match against GlobalAccount regexp
		if rt.GlobalAccount != "" {
			matched, err := regexp.MatchString(rt.GlobalAccount, shoot.Labels[globalAccountLabel])
			if err != nil || !matched {
				continue
			}
		}

		// Perform match against SubAccount regexp
		if rt.SubAccount != "" {
			matched, err := regexp.MatchString(rt.SubAccount, shoot.Labels[subAccountLabel])
			if err != nil || !matched {
				continue
			}
		}

		// Perform match against Region regexp
		if rt.Region != "" {
			matched, err := regexp.MatchString(rt.Region, shoot.Spec.Region)
			if err != nil || !matched {
				continue
			}
		}

		// Check if target: all is specified
		if rt.Target != "" && rt.Target != orchestration.TargetAll {
			continue
		}

		runtimes = append(runtimes, resolver.runtimeFromOperationStatus(instanceOpStatus, shoot.Name, maintenanceWindowBegin, maintenanceWindowEnd))
	}

	return runtimes, nil
}

func (*GardenerRuntimeResolver) runtimeFromOperationStatus(opStatus *instanceOperationStatus, shootName string, windowBegin, windowEnd time.Time) internal.Runtime {
	return internal.Runtime{
		InstanceID:             opStatus.InstanceID,
		RuntimeID:              opStatus.RuntimeID,
		GlobalAccountID:        opStatus.GlobalAccountID,
		SubAccountID:           opStatus.SubAccountID,
		ShootName:              shootName,
		MaintenanceWindowBegin: windowBegin,
		MaintenanceWindowEnd:   windowEnd,
	}
}
