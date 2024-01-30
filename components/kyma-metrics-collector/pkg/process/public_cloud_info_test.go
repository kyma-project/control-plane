package process

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"

	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
)

const (
	providersFile = "../testing/fixtures/static_providers.json"
)

func TestGetFeature(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	providersData, err := kmctesting.LoadFixtureFromFile(providersFile)
	g.Expect(err).Should(gomega.BeNil())
	config := &env.Config{PublicCloudSpecs: string(providersData)}
	providers, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	testCases := []struct {
		cloudProvider   string
		vmType          string
		expectedFeature Feature
	}{
		{
			cloudProvider: "azure",
			vmType:        "standard_a2_v2",
			expectedFeature: Feature{
				CpuCores: 2,
				Memory:   4,
				Storage:  20,
				MaxNICs:  2,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d8_v3",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
				Storage:  200,
				MaxNICs:  4,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D4s_v5",
			expectedFeature: Feature{
				CpuCores: 4,
				Memory:   16,
				Storage:  0,
				MaxNICs:  2,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D8s_v5",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
				Storage:  0,
				MaxNICs:  4,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D16s_v5",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
				Storage:  0,
				MaxNICs:  8,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D32s_v5",
			expectedFeature: Feature{
				CpuCores: 32,
				Memory:   128,
				Storage:  0,
				MaxNICs:  8,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D48s_v5",
			expectedFeature: Feature{
				CpuCores: 48,
				Memory:   192,
				Storage:  0,
				MaxNICs:  8,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "Standard_D64s_v5",
			expectedFeature: Feature{
				CpuCores: 64,
				Memory:   256,
				Storage:  0,
				MaxNICs:  8,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d8_foo",
		},
		{
			cloudProvider: "aws",
			vmType:        "m5.2xlarge",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "aws",
			vmType:        "t4g.nano",
			expectedFeature: Feature{
				CpuCores: 2,
				Memory:   0.5,
			},
		},
		{
			cloudProvider: "aws",
			vmType:        "m5.2xlarge.foo",
		},
		{
			cloudProvider: "gcp",
			vmType:        "n2-standard-8",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "gcp",
			vmType:        "n2-standard-16",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c12_m48",
			expectedFeature: Feature{
				CpuCores: 12,
				Memory:   48,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c16_m64",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c32_m128",
			expectedFeature: Feature{
				CpuCores: 32,
				Memory:   128,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c4_m16",
			expectedFeature: Feature{
				CpuCores: 4,
				Memory:   16,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c64_m256",
			expectedFeature: Feature{
				CpuCores: 64,
				Memory:   256,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c6_m24",
			expectedFeature: Feature{
				CpuCores: 6,
				Memory:   24,
			},
		},
		{
			cloudProvider: "openstack",
			vmType:        "g_c8_m32",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
	}

	for _, tc := range testCases {
		gotFeature := providers.GetFeature(tc.cloudProvider, tc.vmType)
		if gotFeature != nil {
			g.Expect(*gotFeature).Should(gomega.Equal(tc.expectedFeature))
			continue
		}
		g.Expect(gotFeature).Should(gomega.BeNil())
	}
}
