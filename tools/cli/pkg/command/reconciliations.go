package command

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	ErrMothershipResponse = errors.New("reconciler error response")
)

func isErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func responseErr(resp *http.Response) error {
	var msg string
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		msg = "unknown error"
	}
	return errors.Wrapf(ErrMothershipResponse, "%s %d", msg, resp.StatusCode)
}

// TODO: Changes in context of NewReconciliationCmd are imlpementing here - https://github.com/kyma-project/control-plane/pull/931
// NewUpgradeCmd constructs the reconciliation command and all subcommands under the reconciliation command
func NewReconciliationCmd(mothershipURL string) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:     "reconciliations",
		Aliases: []string{"rc"},
		Short:   "Displays Kyma Reconciliations.",
		Long: `Displays Kyma Reconciliations and their primary attributes, such as reconciliation-id.
The command supports filtering Reconciliations based on`,
	}

	cobraCmd.AddCommand(
		NewReconciliationEnableCmd(),
		NewReconciliationDisableCmd(),
	)
	return cobraCmd
}
