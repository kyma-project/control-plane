package gardener

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDNSProvidersValuesFromYAML(t *testing.T) {

	t.Run("should read default DNS Providers values", func(t *testing.T) {
		// given
		dnsTestFilePath := "testdata/dns-values.yaml"

		expectedDNSProvidersValues := internal.DNSProvidersData{
			Providers: []internal.DNSProviderData{
				{
					DomainsInclude: []string{"dev.kyma.ondemand.com"},
					Primary:        true,
					SecretName:     "vv-test-aws-route53-secret",
					Type:           "aws-route53",
				},
			},
		}

		// when
		dnsProvidersValues, err := ReadDNSProvidersValuesFromYAML(dnsTestFilePath)
		t.Logf("%v,%v,%v,%v\n", dnsProvidersValues.Providers[0].DomainsInclude, dnsProvidersValues.Providers[0].Primary, dnsProvidersValues.Providers[0].SecretName, dnsProvidersValues.Providers[0].Type)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDNSProvidersValues, dnsProvidersValues)
	})

	t.Run("should return error while reading YAML file", func(t *testing.T) {
		// given
		nonExistentFilePath := "not/existent/file.yaml"

		// when
		dnsProvidersValues, err := ReadDNSProvidersValuesFromYAML(nonExistentFilePath)

		// then
		require.Error(t, err)
		assert.Equal(t, internal.DNSProvidersData{}, dnsProvidersValues)
	})
}
