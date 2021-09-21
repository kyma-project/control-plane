package command

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	reconciliationScheme = "https"
)

var (
	ErrMothershipResponse = errors.New("reconciler error response")
)

type ReconciliationParams struct {
	RuntimeIDs []string
	Statyses   []mothership.Status
	Shoots     []string
}

func (rp *ReconciliationParams) asMap() map[string]string {
	result := map[string]string{}

	if len(rp.RuntimeIDs) > 0 {
		rtIDs := strings.Join(rp.RuntimeIDs, ",")
		result["runtime-id"] = rtIDs
	}

	if len(rp.Statyses) > 0 {
		var states string
		for i, state := range rp.Statyses {
			separator := ","
			if i == len(rp.Statyses) || i == 0 {
				separator = ""
			}
			states = fmt.Sprintf("%s%s%s", states, separator, state)
		}
		result["state"] = states
	}

	if len(rp.Shoots) > 0 {
		var shoots string
		for i, shoot := range rp.Shoots {
			separator := ","
			if i == len(rp.Shoots) || i == 0 {
				separator = ""
			}
			shoots = fmt.Sprintf("%s%s%s", shoots, separator, shoot)
		}
		result["shoot"] = shoots
	}

	return result
}

type ReconciliationCommand struct {
	ctx           context.Context
	mothershipURL string
	log           logger.Logger
	output        string
	params        ReconciliationParams
	rawStates     []string
}

func validateReconciliationStates(rawStates []string, params *ReconciliationParams) error {
	for _, s := range rawStates {
		val := mothership.Status(strings.Trim(s, " "))
		switch val {
		case mothership.StatusReady, mothership.StatusError, mothership.StatusReconcilePending, mothership.StatusReconciling:
			params.Statyses = append(params.Statyses, val)
		default:
			return fmt.Errorf("invalid value for state: %s", s)
		}
	}
	return nil
}

func (cmd *ReconciliationCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}
	// Validate and transform states
	return validateReconciliationStates(cmd.rawStates, &cmd.params)
}

func (cmd *ReconciliationCommand) printReconciliation(data []mothership.ReconcilerStatus) error {
	switch {
	case cmd.output == tableOutput:
		tp, err := printer.NewTablePrinter([]printer.Column{
			{
				Header:    "ID",
				FieldSpec: "{.ID}",
			},
		}, false)
		if err != nil {
			return err
		}
		return tp.PrintObj(data)
	case cmd.output == jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(data)
	case strings.HasPrefix(cmd.output, customOutput):
		_, templateFile := printer.ParseOutputToTemplateTypeAndElement(cmd.output)
		column, err := printer.ParseColumnToHeaderAndFieldSpec(templateFile)
		if err != nil {
			return err
		}

		ccp, err := printer.NewTablePrinter(column, false)
		if err != nil {
			return err
		}
		return ccp.PrintObj(data)
	}
	return nil
}

func isErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func responseErr(resp *http.Response) error {
	var msg string
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		msg = "unknown error"
	}
	return errors.Wrapf(ErrMothershipResponse, "%s %d", msg, resp.StatusCode)
}

func (cmd *ReconciliationCommand) Run() error {
	cmd.log = logger.New()

	client, err := mothership.NewClient(cmd.mothershipURL)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	response, err := client.GetReconciles(ctx, func(ctx context.Context, req *http.Request) error {
		cmd.params.asMap()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "wile listing reconciliations")
	}

	defer response.Body.Close()

	if isErrResponse(response.StatusCode) {
		err := responseErr(response)
		return err
	}

	var result []mothership.ReconcilerStatus
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return errors.WithStack(ErrMothershipResponse)
	}

	err = cmd.printReconciliation(result)
	if err != nil {
		return errors.Wrap(err, "while printing runtimes")
	}

	return nil
}

// NewUpgradeCmd constructs the reconciliation command and all subcommands under the reconciliation command
func NewReconciliationCmd(mothershipURL string) *cobra.Command {
	cmd := ReconciliationCommand{
		mothershipURL: mothershipURL,
	}

	cobraCmd := &cobra.Command{
		Use:     "reconciliations",
		Aliases: []string{"rc"},
		Short:   "Displays Kyma Reconciliations.",
		Long: `Displays Kyma Reconciliations and their primary attributes, such as reconciliation-id.
The command supports filtering Reconciliations based on`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.params.RuntimeIDs, "runtime-id", "r", nil, "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.rawStates, "state", "S", nil, "Filter by Reconciliation state. The possible values are: ok, err, suspended, all. Suspended Reconciliations are filtered out unless the \"all\" or \"suspended\" values are provided. You can provide multiple values, either separated by a comma (e.g. ok,err), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Shoots, "shoot", "c", nil, "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}
