package orchestration

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerclient "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	brokerapi "github.com/pivotal-cf/brokerapi/v8/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RuntimeLister is the interface to get runtime objects from KEB
//go:generate mockery --name=RuntimeLister --output=. --outpkg=orchestration --case=underscore --structname RuntimeListerMock --filename runtime_lister_mock.go
type RuntimeLister interface {
	ListAllRuntimes() ([]runtime.RuntimeDTO, error)
}

// GardenerRuntimeResolver is the default resolver which implements the RuntimeResolver interface.
// This resolver uses the Shoot resources on the Gardener cluster to resolve the runtime targets.
//
// Naive implementation, listing all the shoots and perfom filtering on the result.
// The logic could be optimized with k8s client cache using shoot lister / indexer.
// The implementation is thread safe, i.e. it is safe to call Resolve() from multiple threads concurrently.
type GardenerRuntimeResolver struct {
	gardenerClient    gardenerclient.CoreV1beta1Interface
	gardenerNamespace string
	runtimeLister     RuntimeLister
	runtimes          map[string]runtime.RuntimeDTO
	mutex             sync.RWMutex
	logger            logrus.FieldLogger
}

const (
	globalAccountLabel      = "account"
	subAccountLabel         = "subaccount"
	runtimeIDAnnotation     = "kcp.provisioner.kyma-project.io/runtime-id"
	maintenanceWindowFormat = "150405-0700"
)

// NewGardenerRuntimeResolver constructs a GardenerRuntimeResolver with the mandatory input parameters.
func NewGardenerRuntimeResolver(gardenerClient gardenerclient.CoreV1beta1Interface, gardenerNamespace string, lister RuntimeLister, logger logrus.FieldLogger) *GardenerRuntimeResolver {
	return &GardenerRuntimeResolver{
		gardenerClient:    gardenerClient,
		gardenerNamespace: gardenerNamespace,
		runtimeLister:     lister,
		runtimes:          map[string]runtime.RuntimeDTO{},
		logger:            logger.WithField("orchestration", "resolver"),
	}
}

// Resolve given an input slice of target specs to include and exclude, returns back a list of unique Runtime objects
func (resolver *GardenerRuntimeResolver) Resolve(targets TargetSpec) ([]Runtime, error) {
	runtimeIncluded := map[string]bool{}
	runtimeExcluded := map[string]bool{}
	runtimes := []Runtime{}
	shoots, err := resolver.getAllShoots()
	if err != nil {
		return nil, errors.Wrapf(err, "while listing gardener shoots in namespace %s", resolver.gardenerNamespace)
	}
	err = resolver.syncRuntimeOperations()
	if err != nil {
		return nil, errors.Wrap(err, "while syncing runtimes")
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
	ctx := context.Background()
	shootList, err := resolver.gardenerClient.Shoots(resolver.gardenerNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return shootList.Items, nil
}

func (resolver *GardenerRuntimeResolver) syncRuntimeOperations() error {
	runtimes, err := resolver.runtimeLister.ListAllRuntimes()
	if err != nil {
		return err
	}
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()

	for _, rt := range runtimes {
		resolver.runtimes[rt.RuntimeID] = rt
	}

	return nil
}

func (resolver *GardenerRuntimeResolver) getRuntime(runtimeID string) (runtime.RuntimeDTO, bool) {
	resolver.mutex.RLock()
	defer resolver.mutex.RUnlock()
	rt, ok := resolver.runtimes[runtimeID]

	return rt, ok
}

func (resolver *GardenerRuntimeResolver) resolveRuntimeTarget(rt RuntimeTarget, shoots []gardenerapi.Shoot) ([]Runtime, error) {
	runtimes := []Runtime{}
	// r.Plan = resolver.runtimes[r.RuntimeID].ServicePlanName
	// Iterate over all shoots. Evaluate target specs. If multiple are specified, all must match for a given shoot.
	for _, shoot := range shoots {
		runtimeID := shoot.Annotations[runtimeIDAnnotation]
		if runtimeID == "" {
			resolver.logger.Errorf("Failed to get runtimeID from %s annotation for Shoot %s", runtimeIDAnnotation, shoot.Name)
			continue
		}
		r, ok := resolver.getRuntime(runtimeID)
		if !ok {
			resolver.logger.Errorf("Couldn't find runtime for runtimeID %s", runtimeID)
			continue
		}

		lastOp, lastOpType := runtime.FindLastOperation(r)
		// Skip runtimes for which the last operation is
		//  - not succeeded provision or unsuspension
		//  - suspension
		//  - deprovision
		if lastOpType == runtime.Deprovision || lastOpType == runtime.Suspension || (lastOpType == runtime.Provision || lastOpType == runtime.Unsuspension) && lastOp.State != string(brokerapi.Succeeded) {
			resolver.logger.Infof("Skipping Shoot %s (runtimeID: %s, instanceID %s) due to %s state: %s", shoot.Name, runtimeID, r.InstanceID, lastOpType, lastOp.State)
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
				runtimes = append(runtimes, resolver.runtimeFromDTO(r, shoot.Name, maintenanceWindowBegin, maintenanceWindowEnd))
			}
			continue
		}

		// Match exact shoot by instanceID
		if rt.InstanceID != "" {
			if rt.InstanceID != r.InstanceID {
				continue
			}
		}

		// Match exact shoot by name
		if rt.Shoot != "" && rt.Shoot != shoot.Name {
			continue
		}

		// Perform match against a specific PlanName
		if rt.PlanName != "" {
			if rt.PlanName != r.ServicePlanName {
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
		if rt.Target != "" && rt.Target != TargetAll {
			continue
		}

		runtimes = append(runtimes, resolver.runtimeFromDTO(r, shoot.Name, maintenanceWindowBegin, maintenanceWindowEnd))
	}

	return runtimes, nil
}

func (*GardenerRuntimeResolver) runtimeFromDTO(runtime runtime.RuntimeDTO, shootName string, windowBegin, windowEnd time.Time) Runtime {
	return Runtime{
		InstanceID:             runtime.InstanceID,
		RuntimeID:              runtime.RuntimeID,
		GlobalAccountID:        runtime.GlobalAccountID,
		SubAccountID:           runtime.SubAccountID,
		Plan:                   runtime.ServicePlanName,
		ShootName:              shootName,
		MaintenanceWindowBegin: windowBegin,
		MaintenanceWindowEnd:   windowEnd,
		MaintenanceDays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday,
			time.Saturday, time.Sunday},
	}
}
