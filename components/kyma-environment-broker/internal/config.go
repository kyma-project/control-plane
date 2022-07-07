package internal

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

type ConfigForPlan struct {
	AdditionalComponents []runtime.KymaComponent `json:"additional-components"`
}
