package command

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
)

type reconciliationEnableOpts struct {
	runtimeID string
	shootName string
	force     bool
}

type reconciliationEnableCmd struct {
	mothershipURL string
	kebURL        string
	kebAuth       oauth2.TokenSource
	ctx           context.Context

	opts reconciliationEnableOpts
}

func NewReconciliationEnableCmd() *cobra.Command {
	cmd := reconciliationEnableCmd{}

	cobraCmd := &cobra.Command{
		Use:     "enable",
		Aliases: []string{"e"},
		Short:   "Enable cluster reconciliation.",
		Long:    `Enable reconciliation for a cluster based on the given parameter such as the ID of the runtime or shoot name.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "s", "", "Shoot cluster name of the specific Kyma Runtime.")
	cobraCmd.Flags().BoolVarP(&cmd.opts.force, "force", "f", false, "Enable cluster reconciliation in the next reconcilation cycle (as soon as possible).")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}

func (cmd *reconciliationEnableCmd) Validate() error {
	cmd.mothershipURL = GlobalOpts.MothershipAPIURL()
	if cmd.opts.shootName != "" {
		cmd.kebURL = GlobalOpts.KEBAPIURL()
		cmd.kebAuth = CLICredentialManager(logger.New())
	}

	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are used in the same time")
	}

	return nil
}

func (cmd *reconciliationEnableCmd) Run() error {
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

	status := mothership.StatusReady
	if cmd.opts.force {
		status = mothership.StatusReconcilePending
	}

	resp, err := client.PutClustersRuntimeIDStatus(
		ctx, cmd.opts.runtimeID,
		mothership.PutClustersRuntimeIDStatusJSONRequestBody{Status: status},
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

func getRuntimeID(ctx context.Context, kebAddress, shootName string, auth oauth2.TokenSource) (string, error) {
	kebClient := runtime.NewClient(ctx, kebAddress, auth)

	listRtResp, err := kebClient.ListRuntimes(runtime.ListParameters{Shoots: []string{shootName}})
	if err != nil {
		return "", err
	}

	if listRtResp.Count == 0 || listRtResp.Count > 1 {
		return "", errors.Errorf("unexpected number of runtimes for shoot \"%s\": %d", shootName, listRtResp.Count)
	}
	return listRtResp.Data[0].RuntimeID, nil
}
