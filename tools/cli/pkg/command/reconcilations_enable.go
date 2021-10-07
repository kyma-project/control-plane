package command

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
)

type reconcilationEnableOpts struct {
	runtimeID string
	shootName string
	force     bool
}

type reconcilationEnableCmd struct {
	ctx context.Context
	log logger.Logger

	opts reconcilationEnableOpts
}

func (cmd *reconcilationEnableCmd) Validate() error {
	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are used in the same time")
	}

	return nil
}

func (cmd *reconcilationEnableCmd) Run() error {
	cmd.log = logger.New()

	mothershipURL := GlobalOpts.MothershipAPIURL()

	client, err := mothership.NewClient(mothershipURL)
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

func NewReconcilationEnableCmd() *cobra.Command {
	cmd := reconcilationEnableCmd{
		log: logger.New(),
	}

	cobraCmd := &cobra.Command{
		Use:     "enable",
		Aliases: []string{"e"},
		Short:   "TODO",
		Long:    `TODO`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "TODO")
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "r", "", "TODO")
	cobraCmd.Flags().BoolVarP(&cmd.opts.force, "force:", "f", false, "TODO")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}
