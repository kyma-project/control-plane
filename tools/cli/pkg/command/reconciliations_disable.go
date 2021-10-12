package command

import (
	"context"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type reconciliationDisableOpts struct {
	runtimeID string
	shootName string
}

type reconciliationDisableCmd struct {
	mothershipURL string
	ctx           context.Context

	opts reconciliationDisableOpts
}

func (cmd *reconciliationDisableCmd) Validate() error {
	cmd.mothershipURL = GlobalOpts.MothershipAPIURL()

	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are provided in the same time")
	}

	return nil
}

func (cmd *reconciliationDisableCmd) Run() error {
	client, err := mothership.NewClient(cmd.mothershipURL)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	// TODO: use shootID or runtimeID
	resp, err := client.PutClustersRuntimeIDStatus(
		ctx, cmd.opts.runtimeID,
		mothership.PutClustersRuntimeIDStatusJSONRequestBody{Status: mothership.StatusReconcileDisabled},
	)
	if err != nil {
		return errors.Wrap(err, "wile updating cluster status")
	}
	defer resp.Body.Close()

	if isErrResponse(resp.StatusCode) {
		err := responseErr(resp)
		return err
	}

	return nil
}

func NewReconciliationDisableCmd() *cobra.Command {
	cmd := reconciliationDisableCmd{}

	cobraCmd := &cobra.Command{
		Use:     "disable",
		Aliases: []string{"d"},
		Short:   "Disable cluster reconciliation.",
		Long:    `Disable reconciliation for a cluster based on the given parameter such as the ID of the runtime or shoot name.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	// cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "r", "", "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}
