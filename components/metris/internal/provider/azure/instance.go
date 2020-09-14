package azure

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
)

func (i *Instance) getComputeMetrics(ctx context.Context, resourceGroupName string, logger log.Logger, vmcaps *vmCapabilities) *Compute {
	var (
		caps   = *vmcaps
		vms    []compute.VirtualMachine
		disks  []compute.Disk
		cpu    uint64
		ram    float64
		err    error
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

	if i.lastEvent.Compute == nil {
		i.lastEvent.Compute = result
	}

	vms, err = i.client.GetVirtualMachines(ctx, resourceGroupName)
	if err != nil {
		logger.Warnf("could not get virtual machines information, using information from last successful event: %s", err)

		result.VMTypes = i.lastEvent.Compute.VMTypes
		result.ProvisionedCpus = i.lastEvent.Compute.ProvisionedCpus
		result.ProvisionedRAMGB = i.lastEvent.Compute.ProvisionedRAMGB
	} else {
		var vmt = make(map[string]uint32)

		for _, vm := range vms {
			vmtype := string(vm.HardwareProfile.VMSize)
			vmt[vmtype]++

			capabilities, ok := caps[vmtype]
			if !ok {
				logger.Errorf("could not get vm capabilities for type %s", vmtype)
			} else {
				cpu, err = strconv.ParseUint(capabilities[capvCPUs], 10, 64)
				if err != nil {
					logger.Errorf("could not get vm capability %s for type %s: %s", capvCPUs, vmtype, err)
				} else {
					result.ProvisionedCpus += uint32(cpu)
				}

				ram, err = strconv.ParseFloat(capabilities[capMemoryGB], 64)
				if err != nil {
					logger.Errorf("could not get vm capability %s for type %s: %s", capMemoryGB, vmtype, err)
				} else {
					result.ProvisionedRAMGB += ram
				}
			}
		}

		for k, v := range vmt {
			result.VMTypes = append(result.VMTypes, VMType{Name: k, Count: v})
		}
	}

	disks, err = i.client.GetDisks(ctx, resourceGroupName)
	if err != nil {
		logger.With("error", err).Warn("could not get disk information, getting information from last successful event")

		result.ProvisionedVolumes = i.lastEvent.Compute.ProvisionedVolumes
	} else {
		result.ProvisionedVolumes.Count = uint32(len(disks))

		for _, disk := range disks {
			result.ProvisionedVolumes.SizeGBTotal += uint32(*disk.DiskSizeGB)
			result.ProvisionedVolumes.SizeGBRounded += uint32(math.Ceil(float64(*disk.DiskSizeGB)/diskSizeFactor) * diskSizeFactor)
		}
	}

	return result
}

func (i *Instance) getNetworkMetrics(ctx context.Context, resourceGroupName string, logger log.Logger) *Networking {
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

	if i.lastEvent.Networking == nil {
		i.lastEvent.Networking = result
	}

	lbs, err = i.client.GetLoadBalancers(ctx, resourceGroupName)
	if err != nil {
		logger.With("error", err).Warn("could not get loadbalancer infornation, getting information from last successful event")

		result.ProvisionedLoadBalancers = i.lastEvent.Networking.ProvisionedLoadBalancers
	} else {
		result.ProvisionedLoadBalancers += uint32(len(lbs))
	}

	vnets, err = i.client.GetVirtualNetworks(ctx, resourceGroupName)
	if err != nil {
		logger.With("error", err).Warn("could not get vnet infornation, getting information from last successful event")

		result.ProvisionedVnets = i.lastEvent.Networking.ProvisionedVnets
	} else {
		result.ProvisionedVnets += uint32(len(vnets))
	}

	publicIPs, err = i.client.GetPublicIPAddresses(ctx, resourceGroupName)
	if err != nil {
		logger.With("error", err).Warn("could not get public ip infornation, getting information from last successful event")

		result.ProvisionedIps = i.lastEvent.Networking.ProvisionedIps
	} else {
		result.ProvisionedIps += uint32(len(publicIPs))
	}

	return result
}

func (i *Instance) getEventHubMetrics(ctx context.Context, pollinterval time.Duration, resourceGroupName string, logger log.Logger) *EventHub {
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

	if i.lastEvent.EventHub == nil {
		i.lastEvent.EventHub = result
	}

	if resourceGroupName == "" {
		logger.Warn("eventhub namespace is empty, getting information from last successful event")

		result = i.lastEvent.EventHub

		return result
	}

	ehns, eherr := i.client.GetEHNamespaces(ctx, resourceGroupName)
	if eherr != nil {
		logger.With("error", eherr).Warn("eventhub namespace error, getting information from last successful event")

		result = i.lastEvent.EventHub
	} else {
		result.NumberNamespaces = uint32(len(ehns))

		interval := PT1M
		if pollinterval == intervalPT5M {
			interval = PT5M
		}

		for _, ns := range ehns {
			resourceURI := *ns.ID

			nsmetric, errs := i.client.GetMetricValues(ctx, resourceURI, string(interval), []string{"IncomingBytes", "OutgoingBytes", "IncomingMessages"}, []string{string(insights.Maximum)})
			if len(errs) > 0 {
				logger.With("errors", errs).Warn("eventhub metric error, getting information from last successful event")

				result = i.lastEvent.EventHub

				if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
					break
				}

				continue
			} else {
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
		}
	}

	return result
}
