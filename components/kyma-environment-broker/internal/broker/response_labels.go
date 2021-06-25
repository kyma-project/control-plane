package broker

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func ResponseLabels(op internal.ProvisioningOperation, instance internal.Instance, brokerURL string, enableKubeconfigLabel bool) map[string]string {
	brokerURL = strings.TrimLeft(brokerURL, "https://")
	brokerURL = strings.TrimLeft(brokerURL, "http://")

	responseLabels := make(map[string]string, 0)
	responseLabels["Name"] = op.ProvisioningParameters.Parameters.Name
	if enableKubeconfigLabel {
		responseLabels["KubeconfigURL"] = fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instance.InstanceID)
	}

	return responseLabels
}
