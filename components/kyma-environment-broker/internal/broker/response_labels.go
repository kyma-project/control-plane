package broker

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const (
	kubeconfigURLKey      = "KubeconfigURL"
	trialExpiryDetailsKey = "Trial expiration details"
	trialDocsKey          = "Trial documentation"
	expireDuration        = time.Hour * 24 * 14
	trialDocsURL          = "https://help.sap.com/docs/"
	notExpiredInfoFormat  = "your cluster will expire %s."
	expiredInfoFormat     = "your cluster has expired, it is not operational and the link to the dashboard is no longer valid." +
		" To create a new trial cluster, follow the link to the trial documentation."
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
	if instance.IsExpired() {
		delete(labels, kubeconfigURLKey)
		labels[trialExpiryDetailsKey] = expiredInfoFormat
		labels[trialDocsKey] = trialDocsURL
	} else {
		if hoursLeft < 0 {
			hoursLeft = 0
		}
		daysLeft := math.Round(hoursLeft / 24)
		switch {
		case daysLeft == 0:
			labels[trialExpiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, "today")
		case daysLeft == 1:
			labels[trialExpiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, "in 1 day")
		default:
			daysLeftNotice := fmt.Sprintf("in %2.f days", daysLeft)
			labels[trialExpiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, daysLeftNotice)
		}
	}

	return labels
}

func calculateHoursLeft(expireTime time.Time) float64 {
	timeLeftUntilExpire := time.Until(expireTime)
	timeLeftUntilExpireRoundedToHours := timeLeftUntilExpire.Round(time.Hour)
	return timeLeftUntilExpireRoundedToHours.Hours()
}
