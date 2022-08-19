package broker

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const (
	kubeconfigURLKey = "KubeconfigURL"
	remainingTimeKey = "Remaining time"
	expireDuration   = time.Hour * 24 * 14
)

func ResponseLabels(op internal.ProvisioningOperation, instance internal.Instance, brokerURL string, enableKubeconfigLabel bool) map[string]string {
	brokerURL = strings.TrimLeft(brokerURL, "https://")
	brokerURL = strings.TrimLeft(brokerURL, "http://")

	responseLabels := make(map[string]string, 0)
	responseLabels["Name"] = op.ProvisioningParameters.Parameters.Name
	if enableKubeconfigLabel {
		responseLabels[kubeconfigURLKey] = fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instance.InstanceID)
	}

	return responseLabels
}

func ResponseLabelsWithExpireInfo(op internal.ProvisioningOperation, instance internal.Instance, brokerURL string, enableKubeconfigLabel bool) map[string]string {
	labels := ResponseLabels(op, instance, brokerURL, enableKubeconfigLabel)

	expireTime := instance.CreatedAt.Add(expireDuration)
	hoursLeft := calculateHoursLeft(expireTime)
	if hoursLeft <= 0 {
		delete(labels, kubeconfigURLKey)
		labels[remainingTimeKey] = "0 days"
	} else {
		daysLeft := math.Round(hoursLeft / 24)
		if daysLeft == 1 {
			labels[remainingTimeKey] = fmt.Sprintf("%2.f day", daysLeft)
		} else {
			labels[remainingTimeKey] = fmt.Sprintf("%2.f days", daysLeft)
		}
	}

	return labels
}

func calculateHoursLeft(expireTime time.Time) float64 {
	timeLeftUntilExpire := time.Until(expireTime)
	timeLeftUntilExpireRoundedToHours := timeLeftUntilExpire.Round(time.Hour)
	return timeLeftUntilExpireRoundedToHours.Hours()
}
