package command

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	reconciler "github.com/kyma-project/control-plane/components/reconciler/pkg"
	client "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
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
	reconcilerURL string
	kebURL        string
	auth          oauth2.TokenSource
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
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "c", "", "Shoot cluster name of the specific Kyma Runtime.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

func (cmd *reconciliationDisableCmd) Validate() error {
	cmd.reconcilerURL = GlobalOpts.MothershipAPIURL()
	cmd.auth = CLICredentialManager(logger.New())

	if cmd.opts.shootName != "" {
		cmd.kebURL = GlobalOpts.KEBAPIURL()
	}

	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id and shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are provided in the same time")
	}

	return nil
}

func (cmd *reconciliationDisableCmd) Run() error {
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, cmd.auth)

	if cmd.opts.shootName != "" {
		var err error
		kebClient := runtime.NewClient(cmd.kebURL, httpClient)
		cmd.opts.runtimeID, err = getRuntimeID(kebClient, cmd.opts.shootName)
		if err != nil {
			return errors.Wrap(err, "while listing runtimes")
		}
	}

	client, err := client.NewClient(cmd.reconcilerURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	resp, err := client.PutClustersRuntimeIDStatus(
		ctx, cmd.opts.runtimeID,
		reconciler.PutClustersRuntimeIDStatusJSONRequestBody{Status: reconciler.StatusReconcileDisabled},
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
