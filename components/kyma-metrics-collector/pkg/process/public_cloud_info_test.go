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

func TestGetFeatures(t *testing.T) {
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
	}

	for _, tc := range testCases {
		gotFeatures := providers.GetFeatures(tc.cloudProvider, tc.vmType)
		if gotFeatures != nil {
			gotFeature := gotFeatures.Feature
			g.Expect(*gotFeature).Should(gomega.Equal(tc.expectedFeature))
			continue
		}
		g.Expect(gotFeatures).Should(gomega.BeNil())
	}
}
