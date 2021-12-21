package reconciler

import (
	"fmt"
	"strings"

	contract "github.com/kyma-incubator/reconciler/pkg/keb"
)

func PrettyFailures(response *contract.HTTPClusterResponse) string {
	var errs []string
	for _, f := range *response.Failures {
		errs = append(errs, fmt.Sprintf("component %v failed: %v", f.Component, f.Reason))
	}
	return strings.Join(errs, ", ")
}
