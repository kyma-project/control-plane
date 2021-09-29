package runtime

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const DNSConfigYAMLFilePath = "testdata/dns.yaml"

func TestReadDNSConfigFromYAML(t *testing.T) {

	t.Run("should read dns config from specified file", func(t *testing.T) {
		// given
		expectedDNSConfig := internal.DNSConfigDTO{
			Domain: "shoot.test.customdomain.com",
			Providers: []*internal.DNSProviderDTO{
				{
					Primary:    true,
					SecretName: "aws-route53-secret",
					Type:       "aws-route53",
				},
			},
		}

		// when
		dnsConfig, err := ReadDNSConfigFromYAML(DNSConfigYAMLFilePath)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDNSConfig, dnsConfig)
	})

	t.Run("should return error while reading YAML file", func(t *testing.T) {
		// given
		nonExistentFilePath := "not/existent/file.yaml"

		// when
		dnsConfig, err := ReadDNSConfigFromYAML(nonExistentFilePath)

		// then
		require.Error(t, err)
		assert.Equal(t, internal.DNSConfigDTO{}, dnsConfig)
	})
}
