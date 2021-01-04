package command

import (
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
}

const (
	inProgress = "in progress"
	succeeded  = "succeeded"
	failed     = "failed"
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
  kcp runtimes --account CA4836781TID000000000123456789  Display all Runtimes of a given global account.`,
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
	switch cmd.output {
	case tableOutput:
		tp, err := printer.NewTablePrinter(tableColumns, false)
		if err != nil {
			return err
		}
		return tp.PrintObj(runtimes.Data)
	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(runtimes)
	}

	return nil
}

func runtimeStatus(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	if rt.Status.Deprovisioning != nil {
		switch rt.Status.Deprovisioning.State {
		case inProgress:
			return "deprovisioning"
		case failed:
			return "failed (deprovision)"
		case succeeded:
			return "deprovisioned"
		}
	}

	upgradeCount := rt.Status.UpgradingKyma.Count
	if upgradeCount > 0 {
		// Take the first upgrade operation, assuming that Data is sorted by CreatedBy DESC.
		switch rt.Status.UpgradingKyma.Data[0].State {
		case inProgress:
			return "upgrading"
		case failed:
			return "failed (upgrade)"
		case succeeded:
			return "succeeded"
		}
	}

	switch rt.Status.Provisioning.State {
	case inProgress:
		return "provisioning"
	case failed:
		return "failed (provision)"
	}

	return "succeeded"
}

func runtimeCreatedAt(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return rt.Status.CreatedAt.Format("2006/01/02 15:04:05")
}
