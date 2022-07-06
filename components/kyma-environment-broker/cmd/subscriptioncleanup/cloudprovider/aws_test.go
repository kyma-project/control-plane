package cloudprovider_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider"
)

func TestGetEC2Instances(t *testing.T) {
	// Test data is from sap-aws-skr-dev-cust-00003 secretbinding from gardener.
	test_secret_data := map[string][]byte{
		"accessKeyID":     []byte("AKIA4ETQQISWL3EDKTJ2"),
		"secretAccessKey": []byte("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"),
	}

	// Act & Assert
	rcleaner, err := cloudprovider.NewAwsResourcesCleaner(test_secret_data)
	if err != nil {
		t.Log(err)
	}

	err = rcleaner.Do()
	if err != nil {
		t.Log(err)
	}
}
