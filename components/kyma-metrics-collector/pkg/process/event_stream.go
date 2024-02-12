package process

import (
	"fmt"
	"math"
	"strings"
	"time"

	gardenerawsv1alpha1 "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardenerazurev1alpha1 "github.com/gardener/gardener-extension-provider-azure/pkg/apis/azure/v1alpha1"
	gardenergcpv1alpha1 "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
	gardeneropenstackv1alpha1 "github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
)

const (
	nodeInstanceTypeLabel = "node.kubernetes.io/instance-type"
	// storageRoundingFactor rounds of storage to 32. E.g. 17 -> 32, 33 -> 64
	storageRoundingFactor = 32

	Azure     = "azure"
	AWS       = "aws"
	GCP       = "gcp"
	OpenStack = "openstack"
)

type EventStream struct {
	KubeConfig string
	Metric     edp.ConsumptionMetrics
}

type Input struct {
	shoot    *gardencorev1beta1.Shoot
	nodeList *corev1.NodeList
	pvcList  *corev1.PersistentVolumeClaimList
	svcList  *corev1.ServiceList
}

func (inp Input) Parse(providers *Providers) (*edp.ConsumptionMetrics, error) {
	if inp.nodeList == nil {
		return nil, fmt.Errorf("no nodes data to compute metrics on")
	}
	if inp.shoot == nil {
		return nil, fmt.Errorf("no shoot data to compute metrics on")
	}

	metric := new(edp.ConsumptionMetrics)
	provisionedCPUs := 0
	provisionedMemory := 0.0
	providerType := inp.shoot.Spec.Provider.Type
	vmTypes := make(map[string]int)

	pvcStorage := int64(0)
	pvcStorageRounded := int64(0)
	volumeCount := 0
	vnets := 0

	for _, node := range inp.nodeList.Items {
		nodeType := node.Labels[nodeInstanceTypeLabel]
		nodeType = strings.ToLower(nodeType)

		// Calculate CPU and Memory
		vmFeature := providers.GetFeature(providerType, nodeType)
		if vmFeature == nil {
			return nil, fmt.Errorf("providerType: %s and nodeType: %s does not exist in the map", providerType, nodeType)
		}
		provisionedCPUs += vmFeature.CpuCores
		provisionedMemory += vmFeature.Memory
		vmTypes[nodeType] += 1
	}

	if inp.pvcList != nil {
		// Calculate storage from PVCs
		for _, pvc := range inp.pvcList.Items {
			if pvc.Status.Phase == corev1.ClaimBound {
				currPVC := getSizeInGB(pvc.Status.Capacity.Storage())
				pvcStorage += currPVC
				pvcStorageRounded += getVolumeRoundedToFactor(currPVC)
				volumeCount += 1
			}
		}
	}

	provisionedIPs := 0
	if inp.svcList != nil {
		// Calculate network related information
		for _, svc := range inp.svcList.Items {
			if svc.Spec.Type == "LoadBalancer" {
				provisionedIPs += 1
			}
		}
	}

	// Calculate vnets(for Azure) or vpc(for AWS)
	if inp.shoot.Spec.Provider.InfrastructureConfig != nil {
		rawExtension := *inp.shoot.Spec.Provider.InfrastructureConfig
		switch inp.shoot.Spec.Provider.Type {

		// Raw extensions varies based on the provider type
		case Azure:
			decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
			infraConfig := &gardenerazurev1alpha1.InfrastructureConfig{}
			err := runtime.DecodeInto(decoder, rawExtension.Raw, infraConfig)
			if err != nil {
				return nil, err
			}
			if infraConfig.Networks.VNet.CIDR != nil {
				vnets += 1
			}
		case AWS:
			decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
			infraConfig := &gardenerawsv1alpha1.InfrastructureConfig{}
			err := runtime.DecodeInto(decoder, rawExtension.Raw, infraConfig)
			if err != nil {
				return nil, err
			}
			if infraConfig.Networks.VPC.CIDR != nil {
				vnets += 1
			}
		case GCP:
			decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
			infraConfig := &gardenergcpv1alpha1.InfrastructureConfig{}
			if err := runtime.DecodeInto(decoder, rawExtension.Raw, infraConfig); err != nil {
				return nil, err
			}
			if infraConfig.Networks.VPC != nil && infraConfig.Networks.VPC.CloudRouter != nil {
				vnets += 1
			}
		case OpenStack:
			decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
			infraConfig := &gardeneropenstackv1alpha1.InfrastructureConfig{}
			if err := runtime.DecodeInto(decoder, rawExtension.Raw, infraConfig); err != nil {
				return nil, err
			}
			if infraConfig.Networks.Router != nil {
				vnets += 1
			}
		default:
			return nil, fmt.Errorf("provider: %s does not match in the system", inp.shoot.Spec.Provider.Type)
		}
	}
	metric.Timestamp = getTimestampNow()
	metric.Compute.ProvisionedCpus = provisionedCPUs
	metric.Compute.ProvisionedRAMGb = provisionedMemory

	metric.Compute.ProvisionedVolumes.SizeGbTotal = pvcStorage
	metric.Compute.ProvisionedVolumes.SizeGbRounded = pvcStorageRounded
	metric.Compute.ProvisionedVolumes.Count = volumeCount

	metric.Networking.ProvisionedIPs = provisionedIPs
	metric.Networking.ProvisionedVnets = vnets

	for vmType, count := range vmTypes {
		metric.Compute.VMTypes = append(metric.Compute.VMTypes, edp.VMType{
			Name:  vmType,
			Count: count,
		})
	}

	return metric, nil
}

// getTimestampNow returns the time now in the format of RFC3339
func getTimestampNow() string {
	return time.Now().Format(time.RFC3339)
}

func getVolumeRoundedToFactor(size int64) int64 {
	return int64(math.Ceil(float64(size)/storageRoundingFactor) * storageRoundingFactor)
}

// getSizeInGB converts any value in binarySI representation to GB
// More info: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go#L31
func getSizeInGB(value *resource.Quantity) int64 {
	// Converting to milli to normalize
	milliVal := value.MilliValue()
	gbUnit := math.Pow(2, 30)

	// Converting back from milli to original
	gVal := int64((float64(milliVal) / float64(gbUnit)) / 1000)
	return gVal
}
