package command

import (
	"context"
	"net/http"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	reconciler "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type reconciliationDebugLogsOpts struct {
	schedulingID string
}

type reconciliationDebugLogsCmd struct {
	reconcilerURL string
	auth          oauth2.TokenSource
	ctx           context.Context
	opts          reconciliationDebugLogsOpts
}

func (cmd *reconciliationDebugLogsCmd) Validate() error {
	cmd.reconcilerURL = GlobalOpts.MothershipAPIURL()
	cmd.auth = CLICredentialManager(logger.New())

	if cmd.opts.schedulingID == "" {
		return errors.New("scheduling id cannot be empty")
	}

	return nil
}

func (cmd *reconciliationDebugLogsCmd) Run() error {
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, cmd.auth)
	client, err := reconciler.NewClient(cmd.reconcilerURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	response, err := client.PutReconciliationsSchedulingIDDebug(ctx, cmd.opts.schedulingID)
	if err != nil {
		return errors.Wrap(err, "while doing PUT request to reconciliation debug endpoint")
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			return errors.New("Reconciliation not found")
		}
		var err error
		mothershipErr, err := mothership.ReadErrResponse(response.Body)
		if err != nil {
			return errors.Wrap(err, "while reading response body")
		}
		return mothershipErr.ToError(response.StatusCode)
	}

	return nil
}

func NewReconciliationDebugLogsCmd() *cobra.Command {
	cmd := reconciliationDebugLogsCmd{}

	cobraCmd := &cobra.Command{
		Use:     "debug",
		Short:   "enable debug logs for a reconciliation",
		Long:    "enable debug logs for all -not in progress- operations that belong to a reconciliation",
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.schedulingID, "scheduling-id", "s", "", "Scheduling ID")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	}

	cmd.ctx = context.Background()
	return cobraCmd
}
