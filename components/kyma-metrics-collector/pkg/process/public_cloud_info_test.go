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
