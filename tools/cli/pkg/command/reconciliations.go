package command

import (
	"context"
	"encoding/json"

	// "fmt"
	"net/http"
	// "strings"

	// mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	// "github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	reconciliationScheme = "https"
)

var (
	ErrMothershipResponse = errors.New("reconciler error response")
)

type ReconciliationCommand struct {
	ctx           context.Context
	mothershipURL string
	log           logger.Logger
	output        string
	// params        mothership.GetReconcilesParams
	rawStatuses *[]string
}

// func validateReconciliationStatuses(rawStates *[]string, params *mothership.GetReconcilesParams) error {
// 	statuses := []mothership.Status{}
// 	if rawStates == nil {
// 		return nil
// 	}
// 	for _, s := range *rawStates {
// 		val := mothership.Status(strings.Trim(s, " "))
// 		switch val {
// 		case mothership.StatusReady, mothership.StatusError, mothership.StatusReconcilePending, mothership.StatusReconciling:
// 			statuses = append(statuses, val)
// 		default:
// 			return fmt.Errorf("invalid value for state: %s", s)
// 		}
// 	}

// 	params.Statuses = &statuses
// 	return nil
// }

// func (cmd *ReconciliationCommand) Validate() error {
// 	err := ValidateOutputOpt(cmd.output)
// 	if err != nil {
// 		return err
// 	}
// 	// Validate and transform states
// 	return validateReconciliationStatuses(cmd.rawStatuses, &cmd.params)
// }

// func (cmd *ReconciliationCommand) printReconciliation(data []mothership.Reconciliation) error {
// 	switch {
// 	case cmd.output == tableOutput:
// 		tp, err := printer.NewTablePrinter([]printer.Column{
// 			{
// 				Header:    "RUNTIME ID",
// 				FieldSpec: "{.runtimeID}",
// 			},
// 			{
// 				Header:    "SHOOT NAME",
// 				FieldSpec: "{.shootName}",
// 			},
// 			{
// 				Header:    "SCHEDULING ID",
// 				FieldSpec: "{.schedulingID}",
// 			},
// 			{
// 				Header:         "CREATED AT",
// 				FieldSpec:      "{.created}",
// 				FieldFormatter: reconciliationCreated,
// 			},
// 			{
// 				Header:         "UPDATED",
// 				FieldSpec:      "{.updated}",
// 				FieldFormatter: reconciliationUpdated,
// 			},
// 			{
// 				Header:    "STATUES",
// 				FieldSpec: "{.status}",
// 			},
// 			{
// 				Header:    "LOCK",
// 				FieldSpec: "{.lock}",
// 			},
// 		}, false)
// 		if err != nil {
// 			return err
// 		}
// 		return tp.PrintObj(data)
// 	case cmd.output == jsonOutput:
// 		jp := printer.NewJSONPrinter("  ")
// 		jp.PrintObj(data)
// 	case strings.HasPrefix(cmd.output, customOutput):
// 		_, templateFile := printer.ParseOutputToTemplateTypeAndElement(cmd.output)
// 		column, err := printer.ParseColumnToHeaderAndFieldSpec(templateFile)
// 		if err != nil {
// 			return err
// 		}

// 		ccp, err := printer.NewTablePrinter(column, false)
// 		if err != nil {
// 			return err
// 		}
// 		return ccp.PrintObj(data)
// 	}
// 	return nil
// }

// func reconciliationCreated(obj interface{}) string {
// 	sr := obj.(mothership.Reconciliation)
// 	return sr.Created.Format("2006/01/02 15:04:05")
// }

// func reconciliationUpdated(obj interface{}) string {
// 	sr := obj.(mothership.Reconciliation)
// 	return sr.Updated.Format("2006/01/02 15:04:05")
// }

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

// func (cmd *ReconciliationCommand) Run() error {
// 	cmd.log = logger.New()

// 	mothershipURL := GlobalOpts.MothershipAPIURL()

// 	client, err := mothership.NewClient(mothershipURL)
// 	if err != nil {
// 		return errors.Wrap(err, "while creating mothership client")
// 	}

// 	ctx, cancel := context.WithCancel(cmd.ctx)
// 	defer cancel()

// 	response, err := client.GetReconciles(ctx, &cmd.params)
// 	if err != nil {
// 		return errors.Wrap(err, "wile listing reconciliations")
// 	}

// 	defer response.Body.Close()

// 	if isErrResponse(response.StatusCode) {
// 		err := responseErr(response)
// 		return err
// 	}

// 	var result []mothership.Reconciliation
// 	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
// 		return errors.WithStack(ErrMothershipResponse)
// 	}

// 	err = cmd.printReconciliation(result)
// 	if err != nil {
// 		return errors.Wrap(err, "while printing runtimes")
// 	}

// 	return nil
// }

// NewUpgradeCmd constructs the reconciliation command and all subcommands under the reconciliation command
func NewReconciliationCmd(mothershipURL string) *cobra.Command {
	cmd := ReconciliationCommand{
		mothershipURL: "http://localhost:8080/v1",
	}

	cobraCmd := &cobra.Command{
		Use:     "reconciliations",
		Aliases: []string{"rc"},
		Short:   "Displays Kyma Reconciliations.",
		Long: `Displays Kyma Reconciliations and their primary attributes, such as reconciliation-id.
The command supports filtering Reconciliations based on`,
		// PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		// RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.AddCommand(
		NewReconcilationEnableCmd(),
		NewReconcilationDisableCmd(),
	)

	SetOutputOpt(cobraCmd, &cmd.output)

	// for _, v := range []struct {
	// 	name, shorthand, usage string
	// 	dst                    *[]string
	// }{
	// 	{
	// 		name:      "runtime-id",
	// 		shorthand: "r",
	// 		usage:     "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.",
	// 		dst:       cmd.params.RuntimeIDs,
	// 	},
	// 	{
	// 		name:      "statuses",
	// 		shorthand: "S",
	// 		usage:     "Filter by Reconciliation state. The possible values are: ok, err, suspended, all. Suspended Reconciliations are filtered out unless the \"all\" or \"suspended\" values are provided. You can provide multiple values, either separated by a comma (e.g. ok,err), or by specifying the option multiple times.",
	// 		dst:       cmd.rawStatuses,
	// 	},
	// 	{
	// 		name:      "shoot",
	// 		shorthand: "c",
	// 		usage:     "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.",
	// 		dst:       cmd.params.Shoots,
	// 	},
	// } {
	// 	slice := make([]string, 0)
	// 	cobraCmd.Flags().StringSliceVarP(&slice, v.name, v.shorthand, nil, v.usage)

	// 	if len(slice) < 1 {
	// 		continue
	// 	}

	// 	v.dst = &slice
	// }

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}
