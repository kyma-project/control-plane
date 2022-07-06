package cloudprovider_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider"
)

func TestGetEC2Instances(t *testing.T) {
	// Before running the test, please search for a development aws account credentials
	test_secret_data := map[string][]byte{
		"accessKeyID":     []byte(""),
		"secretAccessKey": []byte(""),
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
