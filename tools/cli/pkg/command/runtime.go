package command

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RuntimeCommand represents an execution of the kcp runtimes command
type RuntimeCommand struct {
	cobraCmd *cobra.Command
	log      logger.Logger
	output   string
	params   runtime.ListParameters
	state    string
}

const (
	inProgress = "in progress"
	succeeded  = "succeeded"
	failed     = "failed"
)

type operationType string

const (
	provision    operationType = "provision"
	deprovision  operationType = "deprovision"
	upgradeKyma  operationType = "kyma upgrade"
	suspension   operationType = "suspension"
	unsuspension operationType = "unsuspension"
)

var tableColumns = []printer.Column{
	{
		Header:    "GLOBALACCOUNT ID",
		FieldSpec: "{.GlobalAccountID}",
	},
	{
		Header:    "SUBACCOUNT ID",
		FieldSpec: "{.SubAccountID}",
	},
	{
		Header:    "SHOOT",
		FieldSpec: "{.ShootName}",
	},
	{
		Header:    "REGION",
		FieldSpec: "{.ProviderRegion}",
	},
	{
		Header:    "PLAN",
		FieldSpec: "{.ServicePlanName}",
	},
	{
		Header:         "CREATED AT",
		FieldFormatter: runtimeCreatedAt,
	},
	{
		Header:         "STATE",
		FieldFormatter: runtimeStatus,
	},
}

// NewRuntimeCmd constructs a new instance of RuntimeCommand and configures it in terms of a cobra.Command
func NewRuntimeCmd() *cobra.Command {
	cmd := RuntimeCommand{}
	cobraCmd := &cobra.Command{
		Use:     "runtimes",
		Aliases: []string{"runtime", "rt"},
		Short:   "Displays Kyma Runtimes.",
		Long: `Displays Kyma Runtimes and their primary attributes, such as identifiers, region, or states.
The command supports filtering Runtimes based on various attributes. See the list of options for more details.`,
		Example: `  kcp runtimes                                           Display table overview about all Runtimes.
  kcp rt -c c-178e034 -o json                            Display all details about one Runtime identified by a Shoot name in the JSON format.
  kcp runtimes --account CA4836781TID000000000123456789  Display all Runtimes of a given global account.
  kcp runtimes -c bbc3ee7 -o custom="INSTANCE ID:instanceID,SHOOTNAME:shootName"
                                                         Display the custom fields about one Runtime identified by a Shoot name.
  kcp runtimes -o custom="INSTANCE ID:instanceID,SHOOTNAME:shootName,runtimeID:runtimeID,STATUS:{status.provisioning}"
                                                         Display all Runtimes with specific custom fields.
  kcp runtimes -t "failed"                               Display all Runtimes with specific state.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Shoots, "shoot", "c", nil, "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.GlobalAccountIDs, "account", "g", nil, "Filter by global account ID. You can provide multiple values, either separated by a comma (e.g. GAID1,GAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.SubAccountIDs, "subaccount", "s", nil, "Filter by subaccount ID. You can provide multiple values, either separated by a comma (e.g. SAID1,SAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.RuntimeIDs, "runtime-id", "i", nil, "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Regions, "region", "r", nil, "Filter by provider region. You can provide multiple values, either separated by a comma (e.g. westeurope,northeurope), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Plans, "plan", "p", nil, "Filter by service plan name. You can provide multiple values, either separated by a comma (e.g. azure,trial), or by specifying the option multiple times.")
	cobraCmd.Flags().StringVarP(&cmd.state, "state", "t", "", "Filter by state. You can provide one value (e.g. succeeded,failed,provision,deprovision).")

	return cobraCmd
}

// Run executes the runtimes command
func (cmd *RuntimeCommand) Run() error {
	cmd.log = logger.New()
	client := runtime.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))

	rp, err := client.ListRuntimes(cmd.params)
	if err != nil {
		return errors.Wrap(err, "while listing runtimes")
	}
	err = cmd.printRuntimes(rp)
	if err != nil {
		return errors.Wrap(err, "while printing runtimes")
	}

	return nil
}

// Validate checks the input parameters of the runtimes command
func (cmd *RuntimeCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *RuntimeCommand) printRuntimes(runtimes runtime.RuntimesPage) error {
	switch {
	case cmd.output == tableOutput:
		tp, err := printer.NewTablePrinter(tableColumns, false)
		if err != nil {
			return err
		}
		if cmd.state != "" {
			return tp.PrintObjWithFieldFormatterFilter(runtimes.Data, cmd.state)
		}

		return tp.PrintObj(runtimes.Data)
	case cmd.output == jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(runtimes)
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
		if cmd.state != "" {
			return ccp.PrintObjWithFieldFormatterFilter(runtimes.Data, cmd.state)
		}
		return ccp.PrintObj(runtimes.Data)
	}
	return nil
}

func runtimeStatus(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return operationStatusToString(runtime.FindLastOperation(rt))
}

func operationStatusToString(op runtime.Operation, t runtime.OperationType) string {
	switch op.State {
	case succeeded:
		switch t {
		case runtime.Deprovision:
			return "deprovisioned"
		case runtime.Suspension:
			return "suspended"
		}
		return "succeeded"
	case failed:
		return fmt.Sprintf("%s (%s)", "failed", t)
	case inProgress:
		switch t {
		case runtime.Provision:
			return "provisioning"
		case runtime.Unsuspension:
			return "provisioning (unsuspending)"
		case runtime.Deprovision:
			return "deprovisioning"
		case runtime.Suspension:
			return "deprovisioning (suspending)"
		case runtime.UpgradeKyma:
			return "upgrading"
		}
	}

	return "succeeded"
}

func runtimeCreatedAt(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return rt.Status.CreatedAt.Format("2006/01/02 15:04:05")
}
