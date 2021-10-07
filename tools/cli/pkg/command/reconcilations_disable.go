package command

import (
	"context"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type reconcilationDisableOpts struct {
	runtimeID string
	shootName string
	force     bool
}

type reconcilationDisableCmd struct {
	ctx context.Context
	log logger.Logger

	opts reconcilationDisableOpts
}

func (cmd *reconcilationDisableCmd) Validate() error {
	if cmd.opts.runtimeID == "" && cmd.opts.shootName == "" {
		return errors.New("runtime-id or shoot is empty")
	}

	if cmd.opts.runtimeID != "" && cmd.opts.shootName != "" {
		return errors.New("runtime-id and shoot are used in the same time")
	}

	return nil
}

func (cmd *reconcilationDisableCmd) Run() error {
	cmd.log = logger.New()

	mothershipURL := GlobalOpts.MothershipAPIURL()

	client, err := mothership.NewClient(mothershipURL)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	// TODO: use shootID or runtimeID
	resp, err := client.PutClustersClusterStatus(
		ctx, cmd.opts.runtimeID,
		mothership.PutClustersClusterStatusJSONRequestBody{Status: mothership.StatusReconcileDisabled},
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

func NewReconcilationDisableCmd() *cobra.Command {
	cmd := reconcilationDisableCmd{}

	cobraCmd := &cobra.Command{
		Use:     "disable",
		Aliases: []string{"d"},
		Short:   "TODO",
		Long:    `TODO`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "TODO")
	cobraCmd.Flags().StringVarP(&cmd.opts.shootName, "shoot", "r", "", "TODO")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	} else {
		cmd.ctx = context.Background()
	}

	return cobraCmd
}
