package steps

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyLabelsAndAnnotationsForLM Set common labels and annotations for kyma lifecycle manager
func ApplyLabelsAndAnnotationsForLM(object client.Object, operation internal.Operation) {
	l := object.GetLabels()
	if l == nil {
		l = make(map[string]string)
	}
	l["kyma-project.io/instance-id"] = operation.InstanceID
	l["kyma-project.io/runtime-id"] = operation.RuntimeID
	l["kyma-project.io/broker-plan-id"] = operation.ProvisioningParameters.PlanID
	l["kyma-project.io/broker-plan-name"] = broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	l["kyma-project.io/global-account-id"] = operation.GlobalAccountID
	l["kyma-project.io/subaccount-id"] = operation.SubAccountID
	l["kyma-project.io/shoot-name"] = operation.ShootName
	l["kyma-project.io/region"] = operation.Region
	l["operator.kyma-project.io/kyma-name"] = KymaName(operation)
	l["operator.kyma-project.io/managed-by"] = "lifecycle-manager"
	object.SetLabels(l)

	a := object.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a["skr-domain"] = operation.ShootDomain
	object.SetAnnotations(a)
}

func KymaKubeconfigName(operation internal.Operation) string {
	return fmt.Sprintf("kubeconfig-%v", KymaName(operation))
}

func KymaResourceGroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "Kyma"}
}

func KymaName(operation internal.Operation) string {
	if operation.KymaResourceName != "" {
		return operation.KymaResourceName
	}
	return strings.ToLower(operation.RuntimeID)
}
