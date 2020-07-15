package testkit

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestWorker allows construction of Gardener workers
type TestWorker struct {
	worker v1beta1.Worker
}

// NewTestWorker creates TestWorker and returns pointer to it, allowing to pipe the constraints
func (tw *TestWorker) NewTestWorker(name string) {
	tw.worker = v1beta1.Worker{
		Machine: v1beta1.Machine{
			Image: &v1beta1.ShootMachineImage{},
		},
		Volume: &v1beta1.Volume{
			Type: util.StringPtr(""),
		},
		MaxSurge:       util.IntOrStringPtr(intstr.FromInt(0)),
		MaxUnavailable: util.IntOrStringPtr(intstr.FromInt(0)),
		Zones:          []string{},
	}
}

// ToWorker returns TestWorker as *v1beta1.Worker
func (tw *TestWorker) ToWorker(name string) *v1beta1.Worker {
	return &tw.worker
}
