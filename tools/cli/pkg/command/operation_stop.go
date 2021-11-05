package command

import (
	"context"
	"io"
	"net/http"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	reconciler "github.com/kyma-project/control-plane/components/reconciler/pkg/auth"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2/json"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type operationDisableOpts struct {
	correlationID string
	schedulingID  string
}

type operationStopCmd struct {
	reconcilerURL string
	auth          oauth2.TokenSource
	ctx           context.Context
	opts          operationDisableOpts
}

func NewOperationStopCmd() *cobra.Command {
	cmd := operationStopCmd{}

	cobraCmd := &cobra.Command{
		Use:     "stop",
		Short:   "stop not queued reconciliation",
		Long:    "Stop operation which is not queued for reconciliation",
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.opts.correlationID, "correlation-id", "c", "", "Correlation ID")
	cobraCmd.Flags().StringVarP(&cmd.opts.schedulingID, "scheduling-id", "s", "", "Scheduling ID")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
	}

	cmd.ctx = context.Background()
	return cobraCmd
}

func (cmd *operationStopCmd) Validate() error {

	cmd.reconcilerURL = GlobalOpts.MothershipAPIURL()
	cmd.auth = CLICredentialManager(logger.New())

	if cmd.opts.schedulingID == "" {
		return errors.New("scheduling id cannot be empty")
	}

	if cmd.opts.correlationID == "" {
		return errors.New("correlation id cannot be empty")
	}

	return nil
}

func (cmd *operationStopCmd) Run() error {
	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	httpClient := oauth2.NewClient(ctx, cmd.auth)

	httpClient = &http.Client{}

	client, err := reconciler.NewClient(cmd.reconcilerURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	reason := mothership.PostOperationsSchedulingIDCorrelationIDStopJSONRequestBody{Reason: "" +
		"Operation set to DONE manually via KCP CLI"}

	response, err := client.PostOperationsSchedulingIDCorrelationIDStop(ctx, cmd.opts.schedulingID, cmd.opts.correlationID, reason)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		// Error handling here
		var err error
		errMsg, err := readErrorResponse(response.Body)
		if err != nil {
			return errors.Wrap(err, "while reading response body")
		}
		return fetchError(response.StatusCode, errMsg)
	}

	return nil
}

//TODO: move it to some generic package, maybe in components!
func readErrorResponse(reader io.Reader) (mothership.HTTPErrorResponse, error) {
	decoder := json.NewDecoder(reader)
	response := mothership.HTTPErrorResponse{}
	err := decoder.Decode(&response)

	return response, err
}

func fetchError(statusCode int, errMsg mothership.HTTPErrorResponse) error {
	var err error
	switch statusCode {
	case http.StatusForbidden:
		{
			err = errors.Errorf("Operation can't be fullfiled, reason: %s", errMsg.Error)
		}
	case http.StatusInternalServerError:
		{
			err = errors.Errorf("Operation can't be fullfiled by server, reason: %s", errMsg.Error)
		}
	default:
		err = errors.Errorf("Unhandled status code: %d, reason: %s", statusCode, errMsg.Error)
	}

	return err
}
