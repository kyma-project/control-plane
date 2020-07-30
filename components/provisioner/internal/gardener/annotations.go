package gardener

import (
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

const (
	auditLogsAnnotation = "custom.shoot.sapcloud.io/subaccountId"
)

type ProvisioningState string

func (s ProvisioningState) String() string {
	return string(s)
}

type KymaInstallationState string

func (s KymaInstallationState) String() string {
	return string(s)
}

const (
	runtimeIDAnnotation   string = "kcp.provisioner.kyma-project.io/runtime-id"
	operationIDAnnotation string = "kcp.provisioner.kyma-project.io/operation-id"

	legacyRuntimeIDAnnotation   string = "compass.provisioner.kyma-project.io/runtime-id"
	legacyOperationIDAnnotation string = "compass.provisioner.kyma-project.io/operation-id"
)

func annotate(shoot *gardener_types.Shoot, annotation, value string) {
	if shoot.Annotations == nil {
		shoot.Annotations = map[string]string{}
	}

	shoot.Annotations[annotation] = value
}

func getRuntimeId(shoot gardener_types.Shoot) string {
	runtimeID, found := shoot.Annotations[runtimeIDAnnotation]
	if !found {
		return ""
	}

	return runtimeID
}
