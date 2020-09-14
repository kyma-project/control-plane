package azure

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/provider/azure/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testVMCaps = make(vmCapabilities) // [vmtype][capname]capvalue

func getMockClient() Client {
	mockClient := mocks.Client{}

	mockErrFn := func(_ context.Context, n string) error {
		if strings.Contains(n, "error") {
			return fmt.Errorf("error")
		}

		return nil
	}

	mockClient.On("GetDisks", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []compute.Disk) {
		if !strings.Contains(n, "diskempty") && !strings.Contains(n, "diskerror") {
			r = *disklist.Value
		}
		return
	}, mockErrFn)

	mockClient.On("GetVirtualMachines", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []compute.VirtualMachine) {
		if !strings.Contains(n, "vmempty") && !strings.Contains(n, "vmerror") {
			r = *vmlist.Value
		}

		return
	}, mockErrFn)

	mockClient.On("GetLoadBalancers", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []network.LoadBalancer) {
		if !strings.Contains(n, "lbempty") && !strings.Contains(n, "lberror") {
			r = *lblist.Value
		}
		return
	}, mockErrFn)

	mockClient.On("GetVirtualNetworks", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []network.VirtualNetwork) {
		if !strings.Contains(n, "vnetempty") && !strings.Contains(n, "vneterror") {
			r = *netlist.Value
		}
		return
	}, mockErrFn)

	mockClient.On("GetPublicIPAddresses", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []network.PublicIPAddress) {
		if !strings.Contains(n, "ipempty") && !strings.Contains(n, "iperror") {
			r = *iplist.Value
		}
		return
	}, mockErrFn)

	mockClient.On("GetEHNamespaces", context.Background(), mock.AnythingOfType("string")).Return(func(_ context.Context, n string) (r []eventhub.EHNamespace) {
		if !strings.Contains(n, "evnsempty") && !strings.Contains(n, "evnserror") {
			r = *nslist.Value
		}

		if strings.Contains(n, "mverr") {
			r = []eventhub.EHNamespace{
				{
					ID:       to.StringPtr("mverr"),
					Name:     to.StringPtr("ns0"),
					Type:     to.StringPtr("Microsoft.EventHub/namespaces"),
					Location: to.StringPtr("eastus"),
				},
			}
		}

		return
	}, mockErrFn)

	mockClient.On("GetMetricValues", context.Background(), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("[]string")).Return(
		func(_ context.Context, uri, interval string, mnames, agg []string) (r map[string]insights.MetricValue) {
			if !strings.Contains(uri, "mvempty") && !strings.Contains(uri, "mverr") {
				r = make(map[string]insights.MetricValue, 3)
				r["IncomingBytes"] = insights.MetricValue{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(41)}
				r["OutgoingBytes"] = insights.MetricValue{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(12)}
				r["IncomingMessages"] = insights.MetricValue{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(136)}
			}
			return
		},
		func(_ context.Context, uri, interval string, mnames, agg []string) []error {
			errs := make([]error, 0)

			if strings.Contains(uri, "mverr") {
				errs = append(errs, fmt.Errorf("error"))
			}

			return errs
		},
	)

	for _, item := range *skulist.Value {
		testVMCaps[*item.Name] = make(map[string]string)
		for _, v := range *item.Capabilities {
			testVMCaps[*item.Name][*v.Name] = *v.Value
		}
	}

	return &mockClient
}

func TestInstance_getComputeMetrics(t *testing.T) {
	mockclient := getMockClient()
	asserts := assert.New(t)

	type fields struct {
		cluster   *gardener.Cluster
		client    Client
		lastEvent *EventData
	}

	type args struct {
		resourceGroupName string
		vmcaps            *vmCapabilities
	}

	vmtypes := []VMType{
		{Name: string(compute.VirtualMachineSizeTypesStandardA8V2), Count: uint32(1)},
		{Name: string(compute.VirtualMachineSizeTypesStandardD8V3), Count: uint32(1)},
	}

	pvol := ProvisionedVolume{
		SizeGBTotal:   uint32(100),
		SizeGBRounded: uint32(128), // ceil(100/32) * 32
		Count:         2,
	}

	metricsBase := Compute{
		VMTypes:            vmtypes,
		ProvisionedCpus:    uint32(16),
		ProvisionedRAMGB:   float64(48),
		ProvisionedVolumes: pvol,
	}

	metricsNoDisk := Compute{
		VMTypes:            vmtypes,
		ProvisionedCpus:    uint32(16),
		ProvisionedRAMGB:   float64(48),
		ProvisionedVolumes: ProvisionedVolume{},
	}

	metricsNoCaps := Compute{
		VMTypes:            vmtypes,
		ProvisionedCpus:    0,
		ProvisionedRAMGB:   0,
		ProvisionedVolumes: pvol,
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   Compute
	}{
		{
			name:   "normal metrics",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "test-resourcegroup", vmcaps: &testVMCaps},
			want:   metricsBase,
		},
		{
			name:   "with no disk",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "diskempty", vmcaps: &testVMCaps},
			want:   metricsNoDisk,
		},
		{
			name:   "with last event",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{Compute: &metricsBase}},
			args:   args{resourceGroupName: "vmerror", vmcaps: &testVMCaps},
			want:   metricsBase,
		},
		{
			name:   "with no capabilities",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "test-resourcegroup", vmcaps: &vmCapabilities{}},
			want:   metricsNoCaps,
		},
	}

	for _, tt := range tests {
		tt := tt // pin

		t.Run(tt.name, func(t *testing.T) {
			i := &Instance{cluster: tt.fields.cluster, client: tt.fields.client, lastEvent: tt.fields.lastEvent}
			got := i.getComputeMetrics(context.Background(), tt.args.resourceGroupName, noopLogger, tt.args.vmcaps)

			asserts.ElementsMatch(tt.want.VMTypes, got.VMTypes)
			asserts.Equal(tt.want.ProvisionedCpus, got.ProvisionedCpus)
			asserts.Equal(tt.want.ProvisionedRAMGB, got.ProvisionedRAMGB)
			asserts.Equal(tt.want.ProvisionedVolumes, got.ProvisionedVolumes)
		})
	}
}

func TestInstance_getNetworkMetrics(t *testing.T) {
	mockclient := getMockClient()
	asserts := assert.New(t)

	type fields struct {
		cluster   *gardener.Cluster
		client    Client
		lastEvent *EventData
	}

	type args struct {
		resourceGroupName string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Networking
	}{
		{
			name:   "normal metrics",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "test-resourcegroup"},
			want:   &Networking{ProvisionedIps: uint32(2), ProvisionedLoadBalancers: uint32(2), ProvisionedVnets: uint32(2)},
		},
		{
			name:   "with no lb",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "lbempty"},
			want:   &Networking{ProvisionedIps: uint32(2), ProvisionedLoadBalancers: 0, ProvisionedVnets: uint32(2)},
		},
		{
			name:   "with no metrics",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "ipempty,lbempty,vnetempty"},
			want:   &Networking{ProvisionedIps: 0, ProvisionedLoadBalancers: 0, ProvisionedVnets: 0},
		},
		{
			name:   "with error but last event",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{Networking: &Networking{ProvisionedIps: uint32(2), ProvisionedLoadBalancers: uint32(2), ProvisionedVnets: uint32(2)}}},
			args:   args{resourceGroupName: "iperror,lberror,vneterror"},
			want:   &Networking{ProvisionedIps: uint32(2), ProvisionedLoadBalancers: uint32(2), ProvisionedVnets: uint32(2)},
		},
	}

	for _, tt := range tests {
		tt := tt // pin

		t.Run(tt.name, func(t *testing.T) {
			i := &Instance{cluster: tt.fields.cluster, client: tt.fields.client, lastEvent: tt.fields.lastEvent}
			got := i.getNetworkMetrics(context.Background(), tt.args.resourceGroupName, noopLogger)
			asserts.Equal(tt.want, got)
		})
	}
}

func TestInstance_getEventHubMetrics(t *testing.T) {
	mockclient := getMockClient()
	asserts := assert.New(t)

	type fields struct {
		cluster   *gardener.Cluster
		client    Client
		lastEvent *EventData
	}

	type args struct {
		pollinterval      time.Duration
		resourceGroupName string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *EventHub
	}{
		{
			name:   "metric PT5M values",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "test-ehresourcegroup", pollinterval: intervalPT5M},
			want:   &EventHub{NumberNamespaces: uint32(1), IncomingRequestsPT5M: float64(136), MaxOutgoingBytesPT5M: float64(12), MaxIncomingBytesPT5M: float64(41)},
		},
		{
			name:   "metric PT1M values",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "test-ehresourcegroup"},
			want:   &EventHub{NumberNamespaces: uint32(1), IncomingRequestsPT1M: float64(136), MaxOutgoingBytesPT1M: float64(12), MaxIncomingBytesPT1M: float64(41)},
		},
		{
			name:   "no namespace error and no lastevent",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "ehnserror"},
			want:   &EventHub{},
		},
		{
			name:   "no namespace error with lastevent",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{EventHub: &EventHub{NumberNamespaces: uint32(1), IncomingRequestsPT5M: float64(136), MaxOutgoingBytesPT5M: float64(12), MaxIncomingBytesPT5M: float64(41)}}},
			args:   args{resourceGroupName: "ehnserror"},
			want:   &EventHub{NumberNamespaces: uint32(1), IncomingRequestsPT5M: float64(136), MaxOutgoingBytesPT5M: float64(12), MaxIncomingBytesPT5M: float64(41)},
		},
		{
			name:   "metric value error with no lastevent",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: "mverr"},
			want:   &EventHub{NumberNamespaces: uint32(1)},
		},
		{
			name:   "no resource group and no lastevent",
			fields: fields{cluster: testCluster, client: mockclient, lastEvent: &EventData{}},
			args:   args{resourceGroupName: ""},
			want:   &EventHub{},
		},
	}

	for _, tt := range tests {
		tt := tt // pinned

		t.Run(tt.name, func(t *testing.T) {
			i := &Instance{cluster: tt.fields.cluster, client: tt.fields.client, lastEvent: tt.fields.lastEvent}
			got := i.getEventHubMetrics(context.Background(), tt.args.pollinterval, tt.args.resourceGroupName, noopLogger)
			asserts.Equal(tt.want, got)
		})
	}
}
