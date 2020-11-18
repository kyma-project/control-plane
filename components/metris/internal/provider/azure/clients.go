package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/mitchellh/mapstructure"
)

func (a *DefaultAuthConfig) GetAuthConfig(clientID, clientSecret, tenantID, envName string) (autorest.Authorizer, azure.Environment, error) {
	ccc := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	var (
		env azure.Environment
		err error
	)

	if envName != "" {
		env, err = azure.EnvironmentFromName(envName)
		if err != nil {
			return nil, azure.PublicCloud, err
		}
	} else {
		env = azure.PublicCloud
	}

	authz, err := ccc.Authorizer()
	if err != nil {
		return nil, env, err
	}

	return authz, env, nil
}

// decode decodes and map kubernetes secret data into the ClientSecretMap structure.
func (s *ClientSecretMap) decode(secrets map[string][]byte) error {
	var (
		decodedSecrets = make(map[string]string)
	)

	for k, v := range secrets {
		decodedSecrets[k] = string(v)
	}

	err := mapstructure.Decode(decodedSecrets, s)
	if err != nil {
		return err
	}

	return nil
}

// newClient return a new client for a cluster base on the cluster configuration provided.
func newClient(cluster *gardener.Cluster, logger log.Logger, tracelevel int, clientAuthConf AuthConfig) (Client, error) {
	conf := &ClientSecretMap{}

	err := conf.decode(cluster.CredentialData)
	if err != nil {
		return nil, err
	}

	clientID := conf.ClientID
	clientSecret := conf.ClientSecret
	tenantID := conf.TenantID
	subscriptionID := conf.SubscriptionID

	authz, env, err := clientAuthConf.GetAuthConfig(clientID, clientSecret, tenantID, conf.EnvironmentName)
	if err != nil {
		return nil, err
	}

	// Unfortunately the azure sdk does not have a baseclient interface, each type has its own baseclient structure definition.
	// So we can't just copy a base client to another, e.i. compute.BaseClient to resources.BaseClient.
	baseclient := &baseClient{}
	baseclient.BaseURI = env.ResourceManagerEndpoint
	baseclient.SubscriptionID = subscriptionID
	baseclient.Authorizer = authz
	baseclient.RequestInspector = LogRequest(logger, tracelevel)
	baseclient.ResponseInspector = LogResponse(logger, tracelevel)

	resourcesBaseClient := baseclient.createResourcesBaseClient()
	computeBaseClient := baseclient.createComputeBaseClient()
	networkBaseClient := baseclient.createNetworkBaseClient()
	insightsBaseClient := baseclient.createInsightsBaseClient()
	eventhubBaseClient := baseclient.createEventhubBaseClient()

	// free memory
	baseclient = nil

	return &client{
		computeBaseClient:   computeBaseClient,
		networkBaseClient:   networkBaseClient,
		insightsBaseClient:  insightsBaseClient,
		resourcesBaseClient: resourcesBaseClient,
		eventhubBaseClient:  eventhubBaseClient,
	}, nil
}

func (c *baseClient) createResourcesBaseClient() *resources.BaseClient {
	baseclient := resources.New(c.SubscriptionID)
	baseclient.Authorizer = c.Authorizer
	baseclient.BaseURI = c.BaseURI
	baseclient.RequestInspector = c.RequestInspector
	baseclient.ResponseInspector = c.ResponseInspector

	return &baseclient
}

func (c *baseClient) createComputeBaseClient() *compute.BaseClient {
	baseclient := compute.New(c.SubscriptionID)
	baseclient.Authorizer = c.Authorizer
	baseclient.BaseURI = c.BaseURI
	baseclient.RequestInspector = c.RequestInspector
	baseclient.ResponseInspector = c.ResponseInspector

	return &baseclient
}

func (c *baseClient) createNetworkBaseClient() *network.BaseClient {
	baseclient := network.New(c.SubscriptionID)
	baseclient.Authorizer = c.Authorizer
	baseclient.BaseURI = c.BaseURI
	baseclient.RequestInspector = c.RequestInspector
	baseclient.ResponseInspector = c.ResponseInspector

	return &baseclient
}

func (c *baseClient) createInsightsBaseClient() *insights.BaseClient {
	baseclient := insights.New(c.SubscriptionID)
	baseclient.Authorizer = c.Authorizer
	baseclient.BaseURI = c.BaseURI
	baseclient.RequestInspector = c.RequestInspector
	baseclient.ResponseInspector = c.ResponseInspector

	return &baseclient
}

func (c *baseClient) createEventhubBaseClient() *eventhub.BaseClient {
	baseclient := eventhub.New(c.SubscriptionID)
	baseclient.Authorizer = c.Authorizer
	baseclient.BaseURI = c.BaseURI
	baseclient.RequestInspector = c.RequestInspector
	baseclient.ResponseInspector = c.ResponseInspector

	return &baseclient
}

// GetResourceGroup returns the of resource group associated with a SKR and the one for the Event Hub.
func (c *client) GetResourceGroup(ctx context.Context, name, filter string, logger log.Logger) (rg resources.Group, err error) {
	metricfn := collectRequestMetrics("resource", "groups")
	defer metricfn()

	rgclient := resources.GroupsClient{BaseClient: *c.resourcesBaseClient}

	if name != "" {
		rg, err = rgclient.Get(ctx, name)
		if err != nil {
			err = fmt.Errorf("%s: %w", err, ErrResourceGroupNotFound)
		}
	} else if filter != "" {
		var rglist resources.GroupListResultPage

		rglist, err = rgclient.List(ctx, filter, nil)
		if err != nil {
			err = fmt.Errorf("%s: %w", err, ErrResourceGroupNotFound)
		} else {
			rgValues := rglist.Values()

			switch rgLen := len(rgValues); {
			case rgLen == 0:
				err = ErrResourceGroupNotFound
			case rgLen > 1:
				logger.Warnf("found more than one event hub resource group for the same subaccountid, taking the first one")
				fallthrough
			default:
				rg = rgValues[0]
			}
		}
	}

	return rg, err
}

// GetVMResourceSkus returns a list of available vm skus.
func (c *client) GetVMResourceSkus(ctx context.Context, filter string) (result []compute.ResourceSku, err error) {
	metricfn := collectRequestMetrics("compute", "skus")
	defer metricfn()

	var (
		skuList   compute.ResourceSkusResultIterator
		skuClient = compute.ResourceSkusClient{BaseClient: *c.computeBaseClient}
	)

	for skuList, err = skuClient.ListComplete(ctx, filter); skuList.NotDone(); err = skuList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		item := skuList.Value()
		if *item.ResourceType == "virtualMachines" {
			result = append(result, skuList.Value())
		}
	}

	return result, err
}

// GetVirtualMachines returns a list of vm used by a resource group.
func (c *client) GetVirtualMachines(ctx context.Context, rgname string) (result []compute.VirtualMachine, err error) {
	metricfn := collectRequestMetrics("compute", "virtualmachines")
	defer metricfn()

	var (
		vmList   compute.VirtualMachineListResultIterator
		vmClient = compute.VirtualMachinesClient{BaseClient: *c.computeBaseClient}
	)

	for vmList, err = vmClient.ListComplete(ctx, rgname); vmList.NotDone(); err = vmList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		result = append(result, vmList.Value())
	}

	return result, err
}

// GetDisks returns a list of disk (non OS) used by a resource group.
func (c *client) GetDisks(ctx context.Context, rgname string) (result []compute.Disk, err error) {
	metricfn := collectRequestMetrics("compute", "disks")
	defer metricfn()

	var (
		diskList   compute.DiskListIterator
		diskClient = compute.DisksClient{BaseClient: *c.computeBaseClient}
	)

	for diskList, err = diskClient.ListByResourceGroupComplete(ctx, rgname); diskList.NotDone(); err = diskList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		disk := diskList.Value()
		if len(disk.DiskProperties.OsType) == 0 {
			result = append(result, disk)
		}
	}

	return result, err
}

// GetLoadBalancers returns a list of load balancer used by a resource group.
func (c *client) GetLoadBalancers(ctx context.Context, rgname string) (result []network.LoadBalancer, err error) {
	metricfn := collectRequestMetrics("network", "loadbalancers")
	defer metricfn()

	var (
		lbList              network.LoadBalancerListResultIterator
		loadBalancersClient = network.LoadBalancersClient{BaseClient: *c.networkBaseClient}
	)

	for lbList, err = loadBalancersClient.ListComplete(ctx, rgname); lbList.NotDone(); err = lbList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		result = append(result, lbList.Value())
	}

	return result, err
}

// GetVirtualNetworks returns a list of virtual networks used by a resource group.
func (c *client) GetVirtualNetworks(ctx context.Context, rgname string) (result []network.VirtualNetwork, err error) {
	metricfn := collectRequestMetrics("network", "virtualnetworks")
	defer metricfn()

	var (
		vnetList              network.VirtualNetworkListResultIterator
		virtualNetworksClient = network.VirtualNetworksClient{BaseClient: *c.networkBaseClient}
	)

	for vnetList, err = virtualNetworksClient.ListComplete(ctx, rgname); vnetList.NotDone(); err = vnetList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		result = append(result, vnetList.Value())
	}

	return result, err
}

// GetPublicIPAddresses returns a list of public ip used by a resource group.
func (c *client) GetPublicIPAddresses(ctx context.Context, rgname string) (result []network.PublicIPAddress, err error) {
	metricfn := collectRequestMetrics("network", "publicipaddresses")
	defer metricfn()

	var (
		ipList                  network.PublicIPAddressListResultIterator
		publicIPAddressesClient = network.PublicIPAddressesClient{BaseClient: *c.networkBaseClient}
	)

	for ipList, err = publicIPAddressesClient.ListComplete(ctx, rgname); ipList.NotDone(); err = ipList.NextWithContext(ctx) {
		if err != nil {
			return result, err
		}

		result = append(result, ipList.Value())
	}

	return result, err
}

// GetEHNamespaces returns a list of Event Hub namespaces for a resource group.
func (c *client) GetEHNamespaces(ctx context.Context, rgname string) (results []eventhub.EHNamespace, err error) {
	metricfn := collectRequestMetrics("eventhub", "namespaces")
	defer metricfn()

	var (
		nsList   eventhub.EHNamespaceListResultIterator
		nsClient = eventhub.NamespacesClient{BaseClient: *c.eventhubBaseClient}
	)

	for nsList, err = nsClient.ListByResourceGroupComplete(ctx, rgname); nsList.NotDone(); err = nsList.NextWithContext(ctx) {
		if err != nil {
			return nil, err
		}

		results = append(results, nsList.Value())
	}

	return results, err
}

// GetMetricValues returns the specified metric data points for the specified resource ID spanning the last 5 minutes.
func (c *client) GetMetricValues(ctx context.Context, resourceURI, interval string, metricnames, aggregations []string) (map[string]insights.MetricValue, []error) {
	metricfn := collectRequestMetrics("insights", "metrics")
	defer metricfn()

	var (
		results   = make(map[string]insights.MetricValue)
		errors    = make([]error, 0)
		endTime   = time.Now().UTC()
		startTime = endTime.Add(time.Duration(-5) * time.Minute)
		timespan  = fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	)

	if len(aggregations) == 0 {
		aggregations = []string{
			string(insights.Average),
			string(insights.Count),
			string(insights.Maximum),
			string(insights.Minimum),
			string(insights.Total),
		}
	}

	metricsClient := insights.MetricsClient{BaseClient: *c.insightsBaseClient}

	// interval possible values: PT1M, PT5M, PT15M, PT30M, PT1H
	metricsList, err := metricsClient.List(ctx, strings.TrimLeft(resourceURI, "/"), timespan, &interval, strings.Join(metricnames, ","), strings.Join(aggregations, ","), nil, "", "", insights.Data, "")
	if err != nil {
		errors = append(errors, fmt.Errorf("%w: %s", ErrMetricClient, err))
		return results, errors
	}

	if metricsList.Value != nil {
		for _, metric := range *metricsList.Value {
			metricName := *metric.Name.Value
			ts := *metric.Timeseries

			if len(ts) == 0 {
				errors = append(errors, fmt.Errorf("%w: %s", ErrTimeseriesNotFound, fmt.Sprintf("metric %s at target %s", metricName, *metric.ID)))
				continue
			}

			tsdata := *ts[0].Data
			if len(tsdata) == 0 {
				errors = append(errors, fmt.Errorf("%w: %s", ErrTimeseriesDataNotFound, fmt.Sprintf("metric %s at target %s", metricName, *metric.ID)))
				continue
			}

			metricValue := tsdata[len(tsdata)-1]
			results[metricName] = metricValue
		}
	} else {
		errors = append(errors, fmt.Errorf("%w: %s", ErrMetricNotFound, fmt.Sprintf("at URI %s", resourceURI)))
	}

	return results, errors
}
