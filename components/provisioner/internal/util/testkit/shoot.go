package testkit

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestShoot allows construction of custom v1beta1.Shoot for testing purposes
type TestShoot struct {
	shoot *v1beta1.Shoot
}

// NewTestShoot creates TestShoot and returns pointer to it, allowing to pipe the constraints
func NewTestShoot(name string) *TestShoot {
	clientID := "9bd05ed7-a930-44e6-8c79-e6defeb1111"
	groupsClaim := "groups"
	issuerURL := "https://kymatest.accounts400.ondemand.com"
	usernameClaim := "sub"
	usernamePrefix := "-"
	return &TestShoot{
		shoot: &v1beta1.Shoot{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{
					AutoUpdate: &v1beta1.MaintenanceAutoUpdate{},
				},
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{},
				},
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							ClientID:       &clientID,
							GroupsClaim:    &groupsClaim,
							IssuerURL:      &issuerURL,
							SigningAlgs:    []string{"RS256"},
							UsernameClaim:  &usernameClaim,
							UsernamePrefix: &usernamePrefix,
						},
					},
				},
			},
			Status: v1beta1.ShootStatus{
				LastOperation: &v1beta1.LastOperation{},
			},
		},
	}
}

// ToShoot returns TestShoot as *v1beta1.Shoot
func (ts *TestShoot) ToShoot() *v1beta1.Shoot {
	return ts.shoot
}

// InNamespace adds namespace to shoot.ObjectMeta.Namespace
func (ts *TestShoot) InNamespace(namespace string) *TestShoot {
	ts.shoot.ObjectMeta.Namespace = namespace
	return ts
}

// WithKubernetesVersion sets value to shoot.Spec.Kubernetes.Version
func (ts *TestShoot) WithKubernetesVersion(v string) *TestShoot {
	ts.shoot.Spec.Kubernetes.Version = v
	return ts
}

// WithAutoUpdate sets values of shoot.Spec.Maintenance.AutoUpdate KubernetesVersion and MachineImageVersion fields
func (ts *TestShoot) WithAutoUpdate(kubernetes, machine bool) *TestShoot {
	ts.shoot.Spec.Maintenance.AutoUpdate.KubernetesVersion = kubernetes
	ts.shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion = machine
	return ts
}

// WithAutoUpdate sets values of shoot.Spec.Maintenance.AutoUpdate KubernetesVersion and MachineImageVersion fields
func (ts *TestShoot) WithPurpose(purpose string) *TestShoot {
	p := v1beta1.ShootPurpose(purpose)
	ts.shoot.Spec.Purpose = &p
	return ts
}

func (ts *TestShoot) WithExposureClassName(exposureClassName string) *TestShoot {
	ts.shoot.Spec.ExposureClassName = &exposureClassName
	return ts
}

// WithWorkers adds v1beta1 Workers to shoot.Spec.Provider.Workers.
// See also testkit.TestWorker
func (ts *TestShoot) WithWorkers(workers ...v1beta1.Worker) *TestShoot {
	ts.shoot.Spec.Provider.Workers = append(ts.shoot.Spec.Provider.Workers, workers...)
	return ts
}

// WithGeneration sets value of shoot.Generation field
func (ts *TestShoot) WithGeneration(generation int64) *TestShoot {
	ts.shoot.Generation = generation
	return ts
}

// WithObservedGeneration sets value of shoot.Status.ObservedGeneration
func (ts *TestShoot) WithObservedGeneration(generation int64) *TestShoot {
	ts.shoot.Status.ObservedGeneration = generation
	return ts
}

// WithOperationError marks shoot.Status.LastOperation.State as 'Error'
func (ts *TestShoot) WithOperationError() *TestShoot {
	ts.shoot.Status.LastOperation.State = v1beta1.LastOperationStateError
	return ts
}

// WithOperationFailed marks shoot.Status.LastOperation.State as 'Failed'
func (ts *TestShoot) WithOperationFailed() *TestShoot {
	ts.shoot.Status.LastOperation.State = v1beta1.LastOperationStateFailed
	return ts
}

func (ts *TestShoot) WithRateLimitExceededError() *TestShoot {
	codes := make([]gardener_types.ErrorCode, 1)
	codes[0] = gardener_types.ErrorInfraRateLimitsExceeded

	lastError := gardener_types.LastError{Codes: codes}

	lastErrors := make([]gardener_types.LastError, 1)
	lastErrors[0] = lastError
	ts.shoot.Status.LastErrors = lastErrors
	return ts
}

// WithOperationPending marks shoot.Status.LastOperation.State as 'Pending'
func (ts *TestShoot) WithOperationPending() *TestShoot {
	ts.shoot.Status.LastOperation.State = v1beta1.LastOperationStatePending
	return ts
}

// WithOperationProcessing marks shoot.Status.LastOperation.State as 'Processing'
func (ts *TestShoot) WithOperationProcessing() *TestShoot {
	ts.shoot.Status.LastOperation.State = v1beta1.LastOperationStateProcessing
	return ts
}

// WithOperationSucceeded marks shoot.Status.LastOperation.State as 'Succeeded'
func (ts *TestShoot) WithOperationSucceeded() *TestShoot {
	ts.shoot.Status.LastOperation.State = v1beta1.LastOperationStateSucceeded
	return ts
}

// WithOperationNil sets shoot.Status.LastOperation to nil
func (ts *TestShoot) WithOperationNil() *TestShoot {
	ts.shoot.Status.LastOperation = nil
	return ts
}

// WithHibernationState sets shoot.Status.IsHibernated and shoot.Status.Constraints
func (ts *TestShoot) WithHibernationState(hibernationPossible bool, hibernated bool) *TestShoot {
	var condition v1beta1.ConditionStatus
	if hibernationPossible {
		condition = v1beta1.ConditionTrue
	} else {
		condition = v1beta1.ConditionFalse
	}

	hibernationPossibleCondition := v1beta1.Condition{
		Type:   v1beta1.ShootHibernationPossible,
		Status: condition,
	}

	ts.shoot.Status.Constraints = []v1beta1.Condition{hibernationPossibleCondition}
	ts.shoot.Status.IsHibernated = hibernated

	return ts
}
