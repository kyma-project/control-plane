package testkit

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestShoot struct {
	shoot *gardener_types.Shoot
}

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

func (ts *TestShoot) ToShoot() *gardener_types.Shoot {
	return ts.shoot
}

func (ts *TestShoot) WithGeneration(generation int64) *TestShoot {
	ts.shoot.Generation = generation
	return ts
}

func (ts *TestShoot) WithObservedGeneration(generation int64) *TestShoot {
	ts.shoot.Status.ObservedGeneration = generation
	return ts
}

func (ts *TestShoot) WithOperationSucceeded() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateSucceeded
	return ts
}

func (ts *TestShoot) WithOperationFailed() *TestShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateFailed
	return ts
}
