package testkit

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestShoot allows construction of custom gardener_types.Shoot for testing purposes
type TestShoot struct {
	shoot *gardener_types.Shoot
}

// NewTestShoot creates TestShoot and returns pointer to it, allowing to pipe the constraints
func NewTestShoot(name string) *TestShoot {
	return &TestShoot{
		shoot: &gardener_types.Shoot{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: gardener_types.ShootSpec{},
			Status: gardener_types.ShootStatus{
				LastOperation: &gardener_types.LastOperation{},
			},
		},
	}
}

// ToShoot returns TestShoot as *gardener_types.Shoot
func (ts *TestShoot) ToShoot() *gardener_types.Shoot {
	return ts.shoot
}

// WithGeneration adds value to shoot.Generation field
func (ts *TestShoot) WithGeneration(generation int64) *TestShoot {
	ts.shoot.Generation = generation
	return ts
}

// WithObservedGeneration adds value to shoot.Status.ObservedGeneration
func (ts *TestShoot) WithObservedGeneration(generation int64) *TestShoot {
	ts.shoot.Status.ObservedGeneration = generation
	return ts
}

// WithOperationError marks shoot.Status.LastOperation.State as 'Error'
func (ts *TestShoot) WithOperationError() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateError
	return ts
}

// WithOperationFailed marks shoot.Status.LastOperation.State as 'Failed'
func (ts *TestShoot) WithOperationFailed() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateFailed
	return ts
}

// WithOperationPending marks shoot.Status.LastOperation.State as 'Pending'
func (ts *TestShoot) WithOperationPending() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStatePending
	return ts
}

// WithOperationProcessing marks shoot.Status.LastOperation.State as 'Processing'
func (ts *TestShoot) WithOperationProcessing() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateProcessing
	return ts
}

// WithOperationSucceeded marks shoot.Status.LastOperation.State as 'Succeeded'
func (ts *TestShoot) WithOperationSucceeded() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateSucceeded
	return ts
}

// WithOperationNil sets shoot.Status.LastOperation to nil
func (ts *TestShoot) WithOperationNil() *TestShoot {
	ts.shoot.Status.LastOperation = nil
	return ts
}
