package command

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
)

type reconciliationEnableOpts struct {
	runtimeID string
	shootName string
	force     bool
}

type reconciliationEnableCmd struct {
	mothershipURL string
	ctx           context.Context

	opts reconciliationEnableOpts
}

func (cmd *reconciliationEnableCmd) Validate() error {
	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are used in the same time")
	}

	return nil
}

func (cmd *reconciliationEnableCmd) Run() error {
	client, err := mothership.NewClient(cmd.mothershipURL)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	status := mothership.StatusReady
	if cmd.opts.force {
		status = mothership.StatusReconcilePending
	}

	// TODO: use shootID or runtimeID
	resp, err := client.PutClustersClusterStatus(
		ctx, cmd.opts.runtimeID,
		mothership.PutClustersClusterStatusJSONRequestBody{Status: status},
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

func NewReconciliationEnableCmd() *cobra.Command {
	cmd := reconciliationEnableCmd{
		mothershipURL: GlobalOpts.MothershipAPIURL(),
	}

	cobraCmd := &cobra.Command{
		Use:     "enable",
		Aliases: []string{"e"},
		Short:   "Enable cluster reconciliation.",
		Long:    `Enable reconciliation for a cluster based on the given parameter such as the ID of the runtime or shoot name.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	// cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "r", "", "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")
	cobraCmd.Flags().BoolVarP(&cmd.opts.force, "force", "f", false, "TODO: Reconcile cluster as soon as possible.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}
