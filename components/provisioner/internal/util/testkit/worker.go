package testkit

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestWorker allows construction of Gardener workers
type TestWorker struct {
	worker v1beta1.Worker
}

// NewTestWorker creates TestWorker and returns pointer to it, allowing to pipe the constraints
func NewTestWorker(name string) *TestWorker {
	return &TestWorker{worker: v1beta1.Worker{
		Machine: v1beta1.Machine{
			Image: &v1beta1.ShootMachineImage{},
		},
		Volume: &v1beta1.Volume{
			Type: util.StringPtr(""),
		},
		MaxSurge:       util.IntOrStringPtr(intstr.FromInt(0)),
		MaxUnavailable: util.IntOrStringPtr(intstr.FromInt(0)),
		Zones:          []string{},
	}}
}

// ToWorker returns TestWorker as *v1beta1.Worker
func (tw *TestWorker) ToWorker() v1beta1.Worker {
	return tw.worker
}

// WithMachineType sets value of worker.Machine.Type
func (tw *TestWorker) WithMachineType(t string) *TestWorker {
	tw.worker.Machine.Type = t
	return tw
}

// WithVolume sets value of worker.Volume Type and Size
func (tw *TestWorker) WithVolume(vType string, size int) *TestWorker {
	tw.worker.Volume.Type = util.StringPtr(vType)
	tw.worker.Volume.Size = fmt.Sprintf("%dGi", size)
	return tw
}

// WithMinMax sets values of Minimum and Maximum
func (tw *TestWorker) WithMinMax(min, max int32) *TestWorker {
	tw.worker.Minimum = min
	tw.worker.Maximum = max
	return tw
}

// WithMaxSurge sets value of MaxSurge
func (tw *TestWorker) WithMaxSurge(max int) *TestWorker {
	tw.worker.MaxSurge = util.IntOrStringPtr(intstr.FromInt(max))
	return tw
}

// WithMaxUnavailable sets value of MaxUnavailable
func (tw *TestWorker) WithMaxUnavailable(max int) *TestWorker {
	tw.worker.MaxUnavailable = util.IntOrStringPtr(intstr.FromInt(max))
	return tw
}

// WithZones adds zones to Zones
func (tw *TestWorker) WithZones(zones ...string) *TestWorker {
	tw.worker.Zones = append(tw.worker.Zones, zones...)
	return tw
}
