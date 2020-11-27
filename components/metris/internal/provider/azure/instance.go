package azure

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/tracing"
	"go.opencensus.io/trace"
)

func (i *Instance) getComputeMetrics(ctx context.Context, logger log.Logger, vmcaps *vmCapabilities) (*Compute, error) {
	var (
		caps   = *vmcaps
		vms    []compute.VirtualMachine
		disks  []compute.Disk
		cpu    uint64
		ram    float64
		err    error
		vmt    = make(map[string]uint32)
		result = &Compute{
			VMTypes:          make([]VMType, 0),
			ProvisionedCpus:  0,
			ProvisionedRAMGB: 0,
			ProvisionedVolumes: ProvisionedVolume{
				SizeGBTotal:   0,
				SizeGBRounded: 0,
				Count:         0,
			},
		}
	)

	if tracing.IsEnabled() {
		var span *trace.Span

		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/getComputeMetrics")
		defer span.End()

		logger = logger.With("traceID", span.SpanContext().TraceID).With("spanID", span.SpanContext().SpanID)
	}

	if vms, err = i.client.GetVirtualMachines(ctx, i.clusterResourceGroupName, logger); err != nil {
		return nil, err
	}

	for _, vm := range vms {
		vmtype := string(vm.HardwareProfile.VMSize)
		vmt[vmtype]++

		if capabilities, ok := caps[vmtype]; ok {
			if cpu, err = strconv.ParseUint(capabilities[capvCPUs], 10, 64); err == nil {
				result.ProvisionedCpus += uint32(cpu)
			} else {
				return nil, fmt.Errorf("could not get vm capability %s for type %s: %s", capvCPUs, vmtype, err)
			}

			if ram, err = strconv.ParseFloat(capabilities[capMemoryGB], 64); err == nil {
				result.ProvisionedRAMGB += ram
			} else {
				return nil, fmt.Errorf("could not get vm capability %s for type %s: %s", capMemoryGB, vmtype, err)
			}
		} else {
			return nil, fmt.Errorf("could not get vm capabilities for type %s", vmtype)
		}
	}

	for k, v := range vmt {
		result.VMTypes = append(result.VMTypes, VMType{Name: k, Count: v})
	}

	if disks, err = i.client.GetDisks(ctx, i.clusterResourceGroupName, logger); err != nil {
		return nil, err
	}

	result.ProvisionedVolumes.Count = uint32(len(disks))

	for _, disk := range disks {
		result.ProvisionedVolumes.SizeGBTotal += uint32(*disk.DiskSizeGB)
		result.ProvisionedVolumes.SizeGBRounded += uint32(math.Ceil(float64(*disk.DiskSizeGB)/diskSizeFactor) * diskSizeFactor)
	}

	return result, nil
}

func (i *Instance) getNetworkMetrics(ctx context.Context, logger log.Logger) (*Networking, error) {
	var (
		result = &Networking{
			ProvisionedLoadBalancers: 0,
			ProvisionedIps:           0,
			ProvisionedVnets:         0,
		}
		err       error
		lbs       []network.LoadBalancer
		vnets     []network.VirtualNetwork
		publicIPs []network.PublicIPAddress
	)

	if tracing.IsEnabled() {
		var span *trace.Span

		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/getNetworkMetrics")
		defer span.End()

		logger = logger.With("traceID", span.SpanContext().TraceID).With("spanID", span.SpanContext().SpanID)
	}

	if lbs, err = i.client.GetLoadBalancers(ctx, i.clusterResourceGroupName, logger); err != nil {
		return nil, err
	}

	result.ProvisionedLoadBalancers += uint32(len(lbs))

	if vnets, err = i.client.GetVirtualNetworks(ctx, i.clusterResourceGroupName, logger); err != nil {
		return nil, err
	}

	result.ProvisionedVnets += uint32(len(vnets))

	if publicIPs, err = i.client.GetPublicIPAddresses(ctx, i.clusterResourceGroupName, logger); err != nil {
		return nil, err
	}

	result.ProvisionedIps += uint32(len(publicIPs))

	return result, nil
}

func (i *Instance) getEventHubMetrics(ctx context.Context, pollinterval time.Duration, logger log.Logger) (*EventHub, error) {
	var (
		result = &EventHub{
			NumberNamespaces:     0,
			IncomingRequestsPT1M: 0,
			MaxIncomingBytesPT1M: 0,
			MaxOutgoingBytesPT1M: 0,
			IncomingRequestsPT5M: 0,
			MaxIncomingBytesPT5M: 0,
			MaxOutgoingBytesPT5M: 0,
		}
	)

	if tracing.IsEnabled() {
		var span *trace.Span

		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/getEventHubMetrics")
		defer span.End()

		logger = logger.With("traceID", span.SpanContext().TraceID).With("spanID", span.SpanContext().SpanID)
	}

	ehns, eherr := i.client.GetEHNamespaces(ctx, i.eventHubResourceGroupName, logger)
	if eherr != nil {
		return nil, eherr
	}

	result.NumberNamespaces = uint32(len(ehns))

	interval := PT1M
	if pollinterval == intervalPT5M {
		interval = PT5M
	}

	for _, ns := range ehns {
		resourceURI := *ns.ID

		nsmetric, err := i.client.GetMetricValues(ctx, resourceURI, string(interval), []string{"IncomingBytes", "OutgoingBytes", "IncomingMessages"}, []string{string(insights.Maximum)}, logger)
		if err != nil {
			return nil, err
		}

		if interval == PT5M {
			result.IncomingRequestsPT5M += *nsmetric["IncomingMessages"].Maximum
			result.MaxIncomingBytesPT5M += *nsmetric["IncomingBytes"].Maximum
			result.MaxOutgoingBytesPT5M += *nsmetric["OutgoingBytes"].Maximum
		} else {
			result.IncomingRequestsPT1M += *nsmetric["IncomingMessages"].Maximum
			result.MaxIncomingBytesPT1M += *nsmetric["IncomingBytes"].Maximum
			result.MaxOutgoingBytesPT1M += *nsmetric["OutgoingBytes"].Maximum
		}
	}

	return result, nil
}
