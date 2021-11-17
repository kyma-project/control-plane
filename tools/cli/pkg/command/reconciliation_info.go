package command

import (
	"context"
	"encoding/json"
	"strings"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type ReconciliationOperationInfoCommand struct {
	ctx          context.Context
	log          logger.Logger
	output       string
	schedulingID string

	provideMshipClient mothershipClientProvider
}

func (cmd *ReconciliationOperationInfoCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}

	if cmd.schedulingID == "" {
		return errors.New("scheduling-id must not be empty")
	}

	return nil
}

func (cmd *ReconciliationOperationInfoCommand) printReconciliation(data mothership.ReconcilationOperationsOKResponse) error {
	switch {
	case cmd.output == tableOutput:
		tp, err := printer.NewTablePrinter([]printer.Column{
			{
				Header:    "COMPONENT",
				FieldSpec: "{.component}",
			},
			{
				Header:    "CORRELATION_ID",
				FieldSpec: "{.correlationID}",
			},
			{
				Header:    "SCHEDULING_ID",
				FieldSpec: "{.schedulingID}",
			},
			{
				Header:    "PRIORITY",
				FieldSpec: "{.priority}",
			},
			{
				Header:    "STATE",
				FieldSpec: "{.state}",
			},
			{
				Header:         "CREATED AT",
				FieldSpec:      "{.created}",
				FieldFormatter: reconciliationOperationCreated,
			},
			{
				Header:         "UPDATED",
				FieldSpec:      "{.updated}",
				FieldFormatter: reconciliationOperationUpdated,
			},
			{
				Header:    "REASON",
				FieldSpec: "{.reason}",
			},
		}, false)
		if err != nil {
			return err
		}

		operations := []mothership.Operation{}
		if len(*data.Operations) != 0 {
			operations = *data.Operations
		}
		return tp.PrintObj(operations)
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

func reconciliationOperationCreated(obj interface{}) string {
	sr := obj.(mothership.Operation)
	return sr.Created.Format("2006/01/02 15:04:05")

}
func reconciliationOperationUpdated(obj interface{}) string {
	sr := obj.(mothership.Operation)
	return sr.Updated.Format("2006/01/02 15:04:05")
}

func (cmd *ReconciliationOperationInfoCommand) Run() error {
	cmd.log = logger.New()

	ctx, cancel := context.WithCancel(cmd.ctx)
	defer cancel()

	// fetch reconciliations
	auth := CLICredentialManager(cmd.log)
	httpClient := oauth2.NewClient(ctx, auth)
	mothershipURL := GlobalOpts.MothershipAPIURL()

	client, err := cmd.provideMshipClient(mothershipURL, httpClient)
	if err != nil {
		return errors.Wrap(err, "while creating mothership client")
	}

	response, err := client.GetReconciliationsSchedulingIDInfo(ctx, cmd.schedulingID)
	if err != nil {
		return errors.Wrap(err, "wile fetching reconciliation operation info")
	}

	defer response.Body.Close()

	if isErrResponse(response.StatusCode) {
		err := responseErr(response)
		return err
	}

	var result mothership.ReconcilationOperationsOKResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return errors.WithStack(ErrMothershipResponse)
	}

	// We are overwriting the kubeconfig field in order to not display it in output
	result.Cluster.Kubeconfig = "<SENSITIVE_INFORMATION>"

	err = cmd.printReconciliation(result)
	if err != nil {
		return errors.Wrap(err, "while printing runtimes")
	}

	return nil
}

// NewUpgradeCmd constructs the reconciliation command and all subcommands under the reconciliation command
func NewReconciliationOperationInfoCmd() *cobra.Command {
	return newReconciliationOperationInfoCmd(defaultMothershipClientProvider)
}

func newReconciliationOperationInfoCmd(mp mothershipClientProvider) *cobra.Command {
	cmd := ReconciliationOperationInfoCommand{
		provideMshipClient: mp,
	}

	cobraCmd := &cobra.Command{
		Use:     "info",
		Aliases: []string{"i"},
		Short:   "Displays Kyma Reconciliations Information.",
		Long:    `Displays Kyma Reconciliations Information and their primary attributes, such as component, correlation-id or priority.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetOutputOpt(cobraCmd, &cmd.output)

	cobraCmd.Flags().StringVarP(&cmd.schedulingID, "scheduling-id", "i", "", "Scheduling ID of the specific Kyma Reconciliation.")

	if cobraCmd.Parent() != nil && cobraCmd.Parent().Context() != nil {
		cmd.ctx = cobraCmd.Parent().Context()
		return cobraCmd
	}

	cmd.ctx = context.Background()
	return cobraCmd
}
