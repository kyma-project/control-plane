package command

import (
	"context"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type reconciliationDisableOpts struct {
	runtimeID string
	shootName string
}

type reconciliationDisableCmd struct {
	mothershipURL string
	kebURL        string
	kebAuth       oauth2.TokenSource
	ctx           context.Context

	opts reconciliationDisableOpts
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

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "s", "", "Shoot cluster name of the specific Kyma Runtime.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}

func (cmd *reconciliationDisableCmd) Validate() error {
	cmd.mothershipURL = GlobalOpts.MothershipAPIURL()

	if cmd.opts.shootName != "" {
		cmd.kebURL = GlobalOpts.KEBAPIURL()
		cmd.kebAuth = CLICredentialManager(logger.New())
	}

	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are provided in the same time")
	}

	return nil
}

func (cmd *reconciliationDisableCmd) Run() error {
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	if cmd.opts.shootName != "" {
		var err error
		cmd.opts.runtimeID, err = getRuntimeID(ctx, cmd.kebURL, cmd.opts.shootName, cmd.kebAuth)
		if err != nil {
			return errors.Wrap(err, "while listing runtimes")
		}
	}

	client, err := mothership.NewClient(cmd.mothershipURL)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

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
