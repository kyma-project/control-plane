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

type operationPurgeOpts struct {
	runtimeID string
}

type operationPurgeCmd struct {
	reconcilerURL string
	auth          oauth2.TokenSource
	ctx           context.Context
	opts          operationPurgeOpts
}

func NewReconciliationPurgeCmd() *cobra.Command {
	cmd := operationPurgeCmd{}

	cobraCmd := &cobra.Command{
		Use:     "purge",
		Short:   "Purge reconciliations for given runtime ID",
		Long:    "Purge all cluster reconciliations for a specified runtime ID",
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

func (cmd *operationPurgeCmd) Validate() error {

	cmd.reconcilerURL = GlobalOpts.MothershipAPIURL()
	cmd.auth = CLICredentialManager(logger.New())

	if cmd.opts.runtimeID == "" {
		return errors.New("runtime id cannot be empty")
	}

	return nil
}

func (cmd *operationPurgeCmd) Run() error {
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, cmd.auth)
	client, err := reconciler.NewClient(cmd.reconcilerURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	response, err := client.DeleteReconciliationsRuntimeID(ctx, cmd.opts.runtimeID)
	if err != nil {
		return errors.Wrap(err, "while doing DELETE request to reconciliations delete endpoint")
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			return errors.New("Cluster not found")
		}
		var err error
		mthshipErr, err := mothership.ReadErrResponse(response.Body)
		if err != nil {
			return errors.Wrap(err, "while reading response body")
		}
		return mthshipErr.ToError(response.StatusCode)
	}
	return nil
}
