package testkit

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testShoot struct {
	shoot *gardener_types.Shoot
}

func NewTestShoot(name string) *testShoot {
	return &testShoot{
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

func (ts *testShoot) ToShoot() *gardener_types.Shoot {
	return ts.shoot
}

func (ts *testShoot) WithGeneration(generation int64) *testShoot {
	ts.shoot.Generation = generation
	return ts
}

func (ts *testShoot) WithObservedGeneration(generation int64) *testShoot {
	ts.shoot.Status.ObservedGeneration = generation
	return ts
}

func (ts *testShoot) WithOperationSucceeded() *testShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateSucceeded
	return ts
}

func (ts *testShoot) WithOperationFailed() *testShoot {
	ts.shoot.Status.LastOperation.State = gardencorev1beta1.LastOperationStateFailed
	return ts
}
