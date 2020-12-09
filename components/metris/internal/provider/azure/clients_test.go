package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/kyma-project/control-plane/components/metris/internal/provider/azure/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	group = resources.Group{
		ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup"),
		Name:       to.StringPtr("test-resourcegroup"),
		Type:       to.StringPtr("Microsoft.Resources/resourceGroups"),
		Properties: &resources.GroupProperties{ProvisioningState: to.StringPtr("Succeeded")},
		Location:   to.StringPtr("westus"),
	}

	grouplist = resources.GroupListResult{
		Value: &[]resources.Group{
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup1"),
				Name:       to.StringPtr("test-resourcegroup1"),
				Type:       to.StringPtr("Microsoft.Resources/resourceGroups"),
				Properties: &resources.GroupProperties{ProvisioningState: to.StringPtr("Succeeded")},
				Location:   to.StringPtr("westus"),
				Tags: map[string]*string{
					"SubAccountID": to.StringPtr("subaccountid-test1"),
				},
			},
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup2"),
				Name:       to.StringPtr("test-resourcegroup2"),
				Type:       to.StringPtr("Microsoft.Resources/resourceGroups"),
				Properties: &resources.GroupProperties{ProvisioningState: to.StringPtr("Succeeded")},
				Location:   to.StringPtr("westus"),
				Tags: map[string]*string{
					"SubAccountID": to.StringPtr("subaccountid-test2"),
				},
			},
		},
	}

	vmlist = compute.VirtualMachineListResult{
		Value: &[]compute.VirtualMachine{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Compute/virtualMachines/vm0"),
				Name:     to.StringPtr("vm0"),
				Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
				Location: to.StringPtr("westus"),
				VirtualMachineProperties: &compute.VirtualMachineProperties{
					HardwareProfile: &compute.HardwareProfile{
						VMSize: compute.VirtualMachineSizeTypesStandardA8V2,
					},
				},
			},
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Compute/virtualMachines/vm1"),
				Name:     to.StringPtr("vm1"),
				Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
				Location: to.StringPtr("westus"),
				VirtualMachineProperties: &compute.VirtualMachineProperties{
					HardwareProfile: &compute.HardwareProfile{
						VMSize: compute.VirtualMachineSizeTypesStandardD8V3,
					},
				},
			},
		},
	}

	disklist = compute.DiskList{
		Value: &[]compute.Disk{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Compute/disk/disk0"),
				Name:     to.StringPtr("disk0"),
				Type:     to.StringPtr("Microsoft.Compute/disk"),
				Location: to.StringPtr("westus"),
				DiskProperties: &compute.DiskProperties{
					OsType:     compute.Linux,
					DiskSizeGB: to.Int32Ptr(50),
				},
			},
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Compute/disk/disk1"),
				Name:     to.StringPtr("disk1"),
				Type:     to.StringPtr("Microsoft.Compute/disk"),
				Location: to.StringPtr("westus"),
				DiskProperties: &compute.DiskProperties{
					DiskSizeGB: to.Int32Ptr(50),
				},
			},
		},
	}

	lblist = network.LoadBalancerListResult{
		Value: &[]network.LoadBalancer{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/loadBalancers/lb0"),
				Name:     to.StringPtr("lb0"),
				Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
				Location: to.StringPtr("westus"),
			},
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/loadBalancers/lb1"),
				Name:     to.StringPtr("lb1"),
				Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
				Location: to.StringPtr("westus"),
			},
		},
	}

	netlist = network.VirtualNetworkListResult{
		Value: &[]network.VirtualNetwork{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/virtualNetworks/vnet0"),
				Name:     to.StringPtr("vnet0"),
				Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
				Location: to.StringPtr("westus"),
			},
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/virtualNetworks/vnet1"),
				Name:     to.StringPtr("vnet1"),
				Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
				Location: to.StringPtr("westus"),
			},
		},
	}

	iplist = network.PublicIPAddressListResult{
		Value: &[]network.PublicIPAddress{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/publicIPAddresses/ip0"),
				Name:     to.StringPtr("ip0"),
				Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
				Location: to.StringPtr("westus"),
			},
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-resourcegroup/providers/Microsoft.Network/publicIPAddresses/ip1"),
				Name:     to.StringPtr("ip1"),
				Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
				Location: to.StringPtr("westus"),
			},
		},
	}

	skulist = compute.ResourceSkusResult{
		Value: &[]compute.ResourceSku{
			{
				ResourceType: to.StringPtr("virtualMachines"),
				Name:         to.StringPtr("Standard_A8_v2"),
				Tier:         to.StringPtr("Standard"),
				Size:         to.StringPtr("A8_v2"),
				Family:       to.StringPtr("standardAv2Family"),
				Locations:    &[]string{"eastus"},
				Capabilities: &[]compute.ResourceSkuCapabilities{
					{Name: to.StringPtr("vCPUs"), Value: to.StringPtr("8")},
					{Name: to.StringPtr("MemoryGB"), Value: to.StringPtr("16")},
				},
			},
			{
				ResourceType: to.StringPtr("virtualMachines"),
				Name:         to.StringPtr("Standard_D8_v3"),
				Tier:         to.StringPtr("Standard"),
				Size:         to.StringPtr("D8_v3"),
				Family:       to.StringPtr("standardDv3Family"),
				Locations:    &[]string{"eastus"},
				Capabilities: &[]compute.ResourceSkuCapabilities{
					{Name: to.StringPtr("vCPUs"), Value: to.StringPtr("8")},
					{Name: to.StringPtr("MemoryGB"), Value: to.StringPtr("32")},
				},
			},
			{
				ResourceType: to.StringPtr("hostGroups/hosts"),
				Name:         to.StringPtr("EASv4-Type1"),
				Family:       to.StringPtr("standardEASv4Family"),
				Locations:    &[]string{"eastus"},
				Capabilities: &[]compute.ResourceSkuCapabilities{
					{Name: to.StringPtr("vCPUs"), Value: to.StringPtr("96")},
					{Name: to.StringPtr("vCPUsPerCore"), Value: to.StringPtr("2")},
				},
			},
		},
	}

	nslist = eventhub.EHNamespaceListResult{
		Value: &[]eventhub.EHNamespace{
			{
				ID:       to.StringPtr("/subscriptions/test-subscriptionid/resourcegroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0"),
				Name:     to.StringPtr("ns0"),
				Type:     to.StringPtr("Microsoft.EventHub/namespaces"),
				Location: to.StringPtr("eastus"),
			},
		},
	}

	metriclist = insights.Response{
		Timespan:       to.StringPtr("2020-06-19T11:33:33Z/2020-06-19T11:33:33Z"),
		Namespace:      to.StringPtr("Microsoft.EventHub/namespaces"),
		Resourceregion: to.StringPtr("eastus"),
		Interval:       to.StringPtr("PT5M"),
		Value: &[]insights.Metric{
			{
				ID:   to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingBytes"),
				Type: to.StringPtr("Microsoft.Insights/metrics"),
				Name: &insights.LocalizableString{Value: to.StringPtr("IncomingBytes"), LocalizedValue: to.StringPtr("Incoming Bytes")},
				Unit: insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{
					{Data: &[]insights.MetricValue{{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(41)}}},
				},
			},
			{
				ID:   to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/OutgoingBytes"),
				Type: to.StringPtr("Microsoft.Insights/metrics"),
				Name: &insights.LocalizableString{Value: to.StringPtr("OutgoingBytes"), LocalizedValue: to.StringPtr("Outgoing Bytes")},
				Unit: insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{
					{Data: &[]insights.MetricValue{{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(12)}}},
				},
			},
			{
				ID:   to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingMessages"),
				Type: to.StringPtr("Microsoft.Insights/metrics"),
				Name: &insights.LocalizableString{Value: to.StringPtr("IncomingMessages"), LocalizedValue: to.StringPtr("Incoming Messages")},
				Unit: insights.UnitCount,
				Timeseries: &[]insights.TimeSeriesElement{
					{Data: &[]insights.MetricValue{{TimeStamp: &date.Time{Time: time.Now()}, Maximum: to.Float64Ptr(136)}}},
				},
			},
		},
	}

	novalueList = insights.Response{
		Timespan:       to.StringPtr("2020-06-19T11:33:33Z/2020-06-19T11:33:33Z"),
		Namespace:      to.StringPtr("Microsoft.EventHub/namespaces"),
		Resourceregion: to.StringPtr("eastus"),
		Interval:       to.StringPtr("PT5M"),
		Value:          &[]insights.Metric{},
	}

	notsList = insights.Response{
		Timespan:       to.StringPtr("2020-06-19T11:33:33Z/2020-06-19T11:33:33Z"),
		Namespace:      to.StringPtr("Microsoft.EventHub/namespaces"),
		Resourceregion: to.StringPtr("eastus"),
		Interval:       to.StringPtr("PT5M"),
		Value: &[]insights.Metric{
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingBytes"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("IncomingBytes"), LocalizedValue: to.StringPtr("Incoming Bytes")},
				Unit:       insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{},
			},
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/OutgoingBytes"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("OutgoingBytes"), LocalizedValue: to.StringPtr("Outgoing Bytes")},
				Unit:       insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{},
			},
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingMessages"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("IncomingMessages"), LocalizedValue: to.StringPtr("Incoming Messages")},
				Unit:       insights.UnitCount,
				Timeseries: &[]insights.TimeSeriesElement{},
			},
		},
	}

	notsdatalist = insights.Response{
		Timespan:       to.StringPtr("2020-06-19T11:33:33Z/2020-06-19T11:33:33Z"),
		Namespace:      to.StringPtr("Microsoft.EventHub/namespaces"),
		Resourceregion: to.StringPtr("eastus"),
		Interval:       to.StringPtr("PT5M"),
		Value: &[]insights.Metric{
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingBytes"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("IncomingBytes"), LocalizedValue: to.StringPtr("Incoming Bytes")},
				Unit:       insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{}}},
			},
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/OutgoingBytes"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("OutgoingBytes"), LocalizedValue: to.StringPtr("Outgoing Bytes")},
				Unit:       insights.UnitBytes,
				Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{}}},
			},
			{
				ID:         to.StringPtr("/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics/IncomingMessages"),
				Type:       to.StringPtr("Microsoft.Insights/metrics"),
				Name:       &insights.LocalizableString{Value: to.StringPtr("IncomingMessages"), LocalizedValue: to.StringPtr("Incoming Messages")},
				Unit:       insights.UnitCount,
				Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{}}},
			},
		},
	}
)

func setupMockAuthConfig(t *testing.T, customURL string) *mocks.AuthConfig {
	t.Helper()

	mockAuthConfig := mocks.AuthConfig{}
	mockAuthConfig.On("GetAuthConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(
		&autorest.NullAuthorizer{},
		func(clientID, clientSecret, tenantID, envName string) azure.Environment {
			env := azure.PublicCloud

			if newenv, err := azure.EnvironmentFromName(envName); err == nil {
				env = newenv
			}

			if customURL != "" {
				env.ResourceManagerEndpoint = customURL
			}

			return env
		},
		func(clientID, clientSecret, tenantID, envName string) error {
			if clientID == "" || clientSecret == "" || tenantID == "" {
				return fmt.Errorf("missing client credentials")
			}

			if envName != "" {
				_, err := azure.EnvironmentFromName(envName)
				if err != nil {
					return fmt.Errorf("no cloud environment matching name %q", envName)
				}
			}

			return nil
		},
	)

	return &mockAuthConfig
}

func urlEscape(format string, params ...interface{}) string {
	uri := fmt.Sprintf(format, params...)

	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	query := u.Query().Encode()
	if len(query) > 0 {
		query = "?" + query
	}

	return u.Path + query
}

// marshalJSON overrides the marshaljson from the original object because it skip some fields
func marshalJSON(obj interface{}) map[string]interface{} {
	objectMap := make(map[string]interface{})
	t := reflect.TypeOf(obj)
	v := reflect.New(t).Elem()

	for i := 0; i < v.NumField(); i++ {
		fld := reflect.ValueOf(obj).Field(i).Interface()
		if fld != nil {
			jsontags := t.Field(i).Tag.Get("json")
			if jsontags != "" && jsontags != "-" {
				fldname := reflect.ValueOf(obj).Type().Field(i).Name
				if fldname == "Value" {
					var subfld []interface{}

					s := reflect.ValueOf(obj).Field(i).Elem()
					for i := 0; i < s.Len(); i++ {
						subfld = append(subfld, marshalJSON(s.Index(i).Interface()))
					}

					fld = subfld
				}

				fldjsonname := strings.Split(jsontags, ",")[0]
				objectMap[fldjsonname] = fld
			}
		}
	}

	return objectMap
}

func newTestServer(responses map[string]interface{}) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "throttling") {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("x-ms-ratelimit-remaining-resource", "Microsoft.Compute/HighCostGet3Min;46")
			w.Header().Add("x-ms-ratelimit-remaining-resource", "Microsoft.Compute/HighCostGet30Min;0")
			w.Header().Add("Retry-After", "2") // wait 2s

			w.WriteHeader(http.StatusTooManyRequests)

			_, err := w.Write([]byte(`{"code": "OperationNotAllowed", "message": "throttling"}`))
			if err != nil {
				return
			}

			return
		}

		// have to remove timestamp and api-version from url because values keep changing
		re := regexp.MustCompile(`(\?|&)(timespan|api-version)=[^&]*`)
		response, ok := responses[string(re.ReplaceAll([]byte(r.RequestURI), []byte("")))]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bodymap := marshalJSON(response)

		body, err := json.MarshalIndent(bodymap, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")

		_, err = w.Write(body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))

	return s
}

func TestDefaultAuthConfig_GetAuthConfig(t *testing.T) {
	var (
		asserts      = assert.New(t)
		clientID     = "clientID"
		clientSecret = "clientSecret"
		tenantID     = "tenantID"
		envName      = ""
	)

	t.Run("default auth config with default environment", func(t *testing.T) {
		authConfig := &DefaultAuthConfig{}
		authz, env, err := authConfig.GetAuthConfig(clientID, clientSecret, tenantID, envName)
		asserts.NoError(err, "should not get an error")
		asserts.Equal(azure.PublicCloud, env)
		asserts.IsType((*autorest.BearerAuthorizer)(nil), authz)
	})

	t.Run("default auth config with bad environment", func(t *testing.T) {
		authConfig := &DefaultAuthConfig{}
		_, _, err := authConfig.GetAuthConfig(clientID, clientSecret, tenantID, "envName")
		asserts.Error(err, "should get an error")
	})

	t.Run("default auth config with no client params", func(t *testing.T) {
		authConfig := &DefaultAuthConfig{}
		_, _, err := authConfig.GetAuthConfig("", "", "", "")
		asserts.Error(err, "should get an error")
	})

	t.Run("auth config with empty base URI", func(t *testing.T) {
		authConfig := setupMockAuthConfig(t, "")
		_, _, err := authConfig.GetAuthConfig(clientID, clientSecret, tenantID, envName)
		asserts.NoError(err, "should not get an error")
	})

	t.Run("auth config with no client params", func(t *testing.T) {
		authConfig := setupMockAuthConfig(t, "http://127.0.0.1")
		_, _, err := authConfig.GetAuthConfig("", "", tenantID, envName)
		asserts.Error(err, "should get an error")
	})

	t.Run("auth config", func(t *testing.T) {
		authConfig := setupMockAuthConfig(t, "http://10.0.0.1")

		expectedEnv := azure.PublicCloud
		expectedEnv.ResourceManagerEndpoint = "http://10.0.0.1"

		authz, env, err := authConfig.GetAuthConfig(clientID, clientSecret, tenantID, envName)
		asserts.NoError(err, "should not get an error")
		asserts.Equal(expectedEnv, env)
		asserts.IsType((*autorest.NullAuthorizer)(nil), authz)
	})
}

func Test_newClient(t *testing.T) {
	asserts := assert.New(t)
	authConfig := setupMockAuthConfig(t, "http://10.0.0.1")

	_, err := newClient(testCluster, noopLogger, authConfig)
	asserts.NoError(err)
}

func Test_clients(t *testing.T) {
	asserts := assert.New(t)

	baseuri := "/subscriptions/test-subscriptionid"
	rgbaseuri := fmt.Sprintf("%s/resourceGroups/test-resourcegroup", baseuri)
	ehrgbaseuri := fmt.Sprintf("%s/resourceGroups/test-ehresourcegroup", baseuri)
	rgfilter := fmt.Sprintf("tagname eq '%s' and tagvalue eq '%s'", tagNameSubAccountID, "subaccountid-test2")
	skufilter := fmt.Sprintf("location eq '%s'", "eastus")

	s := newTestServer(map[string]interface{}{
		urlEscape("%s/resourcegroups?$filter=%s", baseuri, rgfilter):                    grouplist,
		urlEscape("%s/resourcegroups/test-resourcegroup", baseuri):                      group,
		urlEscape("%s/providers/Microsoft.Compute/virtualMachines", rgbaseuri):          vmlist,
		urlEscape("%s/providers/Microsoft.Compute/disks", rgbaseuri):                    disklist,
		urlEscape("%s/providers/Microsoft.Network/loadBalancers", rgbaseuri):            lblist,
		urlEscape("%s/providers/Microsoft.Network/virtualNetworks", rgbaseuri):          netlist,
		urlEscape("%s/providers/Microsoft.Network/publicIPAddresses", rgbaseuri):        iplist,
		urlEscape("%s/providers/Microsoft.EventHub/namespaces", ehrgbaseuri):            nslist,
		urlEscape("%s/providers/Microsoft.Compute/skus?$filter=%s", baseuri, skufilter): skulist,
	})
	defer s.Close()

	authConfig := setupMockAuthConfig(t, s.URL)

	client, cerr := newClient(testCluster, noopLogger, authConfig)
	asserts.NoError(cerr)

	tests := []struct {
		name           string
		ctype          string
		rgname         string
		filter         string
		wants          []interface{}
		wantAssertions []assert.ComparisonAssertionFunc
		errAssertion   assert.ErrorAssertionFunc
	}{
		{
			name:           "get resource group",
			ctype:          "ResourceGroup",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{1},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:         "get ressource group with error",
			ctype:        "ResourceGroup",
			rgname:       "unknownname",
			errAssertion: assert.Error,
		},
		{
			name:           "get ressource group by tag",
			ctype:          "ResourceGroup",
			filter:         fmt.Sprintf("tagname eq '%s' and tagvalue eq '%s'", tagNameSubAccountID, "subaccountid-test2"),
			wants:          []interface{}{1},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:         "get ressource group by tag with error",
			ctype:        "ResourceGroup",
			filter:       fmt.Sprintf("tagname eq '%s' and tagvalue eq '%s'", tagNameSubAccountID, "subaccountid-test3"),
			errAssertion: assert.Error,
		},
		{
			name:           "get vm",
			ctype:          "VirtualMachine",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{2},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:         "get vm with error",
			ctype:        "VirtualMachine",
			rgname:       "fakeResourceGroupName",
			errAssertion: assert.Error,
		},
		{
			name:         "get vm with throttling",
			ctype:        "VirtualMachine",
			rgname:       "throttling",
			errAssertion: assert.Error,
		},
		{
			name:         "get vm with timeout",
			ctype:        "VirtualMachine",
			rgname:       "timeout",
			errAssertion: assert.Error,
		},
		{
			name:           "get disks",
			ctype:          "Disk",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{1, "disk1"},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal, assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:           "get LB",
			ctype:          "LoadBalancer",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{2},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:           "get vnet",
			ctype:          "VirtualNetwork",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{2},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:           "get public ip",
			ctype:          "PublicIPAddress",
			rgname:         "test-resourcegroup",
			wants:          []interface{}{2},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:           "get eh namespaces",
			ctype:          "EHNamespace",
			rgname:         "test-ehresourcegroup",
			wants:          []interface{}{1},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
		{
			name:           "get resource sku",
			ctype:          "ResourceSku",
			filter:         skufilter,
			wants:          []interface{}{2},
			wantAssertions: []assert.ComparisonAssertionFunc{assert.Equal},
			errAssertion:   assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt // pin

		t.Run(tt.name, func(t *testing.T) {
			var (
				err       error
				obj       interface{}
				gotValues []interface{}
			)

			switch tt.ctype {
			case "ResourceGroup":
				obj, err = client.GetResourceGroup(context.Background(), tt.rgname, tt.filter, noopLogger)
				o, ok := obj.(resources.Group)
				if ok {
					l := 0
					n := ""
					if o.Name != nil {
						l = 1
						n = *o.Name
					}
					gotValues = append(gotValues, l, n)
				}

			case "VirtualMachine":
				ctx := context.Background()
				if tt.rgname == "timeout" {
					newctx, cancel := context.WithTimeout(ctx, time.Nanosecond)
					ctx = newctx
					defer cancel()
				}
				obj, err = client.GetVirtualMachines(ctx, tt.rgname, noopLogger)
				gotValues = append(gotValues, len(obj.([]compute.VirtualMachine)))

			case "Disk":
				obj, err = client.GetDisks(context.Background(), tt.rgname, noopLogger)
				o, ok := obj.([]compute.Disk)
				if ok {
					gotValues = append(gotValues, len(o), *o[0].Name)
				}

			case "LoadBalancer":
				obj, err = client.GetLoadBalancers(context.Background(), tt.rgname, noopLogger)
				gotValues = append(gotValues, len(obj.([]network.LoadBalancer)))

			case "VirtualNetwork":
				obj, err = client.GetVirtualNetworks(context.Background(), tt.rgname, noopLogger)
				gotValues = append(gotValues, len(obj.([]network.VirtualNetwork)))

			case "PublicIPAddress":
				obj, err = client.GetPublicIPAddresses(context.Background(), tt.rgname, noopLogger)
				gotValues = append(gotValues, len(obj.([]network.PublicIPAddress)))

			case "EHNamespace":
				obj, err = client.GetEHNamespaces(context.Background(), tt.rgname, noopLogger)
				gotValues = append(gotValues, len(obj.([]eventhub.EHNamespace)))

			case "ResourceSku":
				obj, err = client.GetVMResourceSkus(context.Background(), tt.filter, noopLogger)
				gotValues = append(gotValues, len(obj.([]compute.ResourceSku)))
			}

			tt.errAssertion(t, err)

			if tt.wantAssertions != nil && len(tt.wantAssertions) > 0 {
				for i, assertionFn := range tt.wantAssertions {
					var got interface{}
					if len(gotValues) >= i {
						got = gotValues[i]
					}
					assertionFn(t, tt.wants[i], got)
				}
			}
		})
	}
}

func Test_client_getMetricValues(t *testing.T) {
	asserts := assert.New(t)

	rguri := "/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0/providers/microsoft.insights/metrics"
	params1 := "aggregation=Maximum&interval=PT5M&metricnames=IncomingBytes,OutgoingBytes,IncomingMessages&resultType=Data"
	novalueparams := "aggregation=Maximum&interval=PT5M&metricnames=NoValueMetric&resultType=Data"
	notsparams := "aggregation=Maximum&interval=PT5M&metricnames=NoTSMetric&resultType=Data"
	notsdataparams := "aggregation=Maximum&interval=PT5M&metricnames=NoTSDataMetric&resultType=Data"

	s := newTestServer(map[string]interface{}{
		urlEscape("%s?%s", rguri, params1):        metriclist,
		urlEscape("%s?%s", rguri, novalueparams):  novalueList,
		urlEscape("%s?%s", rguri, notsparams):     notsList,
		urlEscape("%s?%s", rguri, notsdataparams): notsdatalist,
	}components/metris/internal/provider/azure/instance.go)
	defer s.Close()

	authConfig := setupMockAuthConfig(t, s.URL)

	client, cerr := newClient(testCluster, noopLogger, authConfig)
	asserts.NoError(cerr)

	resourceURI := "/subscriptions/test-subscriptionid/resourceGroups/test-ehresourcegroup/providers/Microsoft.EventHub/namespaces/ns0"
	metricnames := []string{"IncomingBytes", "OutgoingBytes", "IncomingMessages"}
	interval := "PT5M"
	aggregations := []string{string(insights.Maximum)}

	t.Run("get maximum metric values", func(t *testing.T) {
		obj, err := client.GetMetricValues(context.Background(), resourceURI, interval, metricnames, aggregations, noopLogger)
		asserts.NoError(err)
		asserts.EqualValues(41, *obj["IncomingBytes"].Maximum)
		asserts.EqualValues(12, *obj["OutgoingBytes"].Maximum)
		asserts.EqualValues(136, *obj["IncomingMessages"].Maximum)
	})

	t.Run("get metric not found error", func(t *testing.T) {
		_, err := client.GetMetricValues(context.Background(), resourceURI, interval, []string{"UnknownMetric"}, aggregations, noopLogger)
		asserts.Error(err)
		asserts.True(errors.Is(err, ErrMetricClient))
	})

	t.Run("get metric with no value error", func(t *testing.T) {
		_, err := client.GetMetricValues(context.Background(), resourceURI, interval, []string{"NoValueMetric"}, aggregations, noopLogger)
		asserts.Error(err)
		asserts.True(errors.Is(err, ErrMetricNotFound))
	})

	t.Run("get metric with no timeseries error", func(t *testing.T) {
		_, err := client.GetMetricValues(context.Background(), resourceURI, interval, []string{"NoTSMetric"}, aggregations, noopLogger)
		asserts.Error(err)
		asserts.True(errors.Is(err, ErrTimeseriesNotFound))
	})

	t.Run("get metric with no timeserie data error", func(t *testing.T) {
		_, err := client.GetMetricValues(context.Background(), resourceURI, interval, []string{"NoTSDataMetric"}, aggregations, noopLogger)
		asserts.Error(err)
		asserts.True(errors.Is(err, ErrTimeseriesDataNotFound))
	})
}
