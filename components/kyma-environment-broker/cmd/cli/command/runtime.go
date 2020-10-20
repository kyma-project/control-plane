package command

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/printer"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RuntimeCommand represents an execution of the kcp runtimes command
type RuntimeCommand struct {
	log    logger.Logger
	output string
	params runtime.ListParameters
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
		Header:         "CREATED AT",
		FieldFormatter: runtimeCreatedAt,
	},
	{
		Header:         "STATE",
		FieldFormatter: runtimeStatus,
	},
}

// NewRuntimeCmd constructs a new instance of RuntimeCommand and configures it in terms of a cobra.Command
func NewRuntimeCmd(log logger.Logger) *cobra.Command {
	cmd := RuntimeCommand{log: log}
	cobraCmd := &cobra.Command{
		Use:     "runtimes",
		Aliases: []string{"runtime", "rt"},
		Short:   "Display Kyma runtimes",
		Long: `Display Kyma runtimes and their primary attributes, such as identifiers, region, states, etc.
The command supports filtering runtimes based on various attributes, see the list of options below.`,
		Example: `  kcp runtimes                                           Display table overview about all runtimes
  kcp rt -c c-178e034 -o json                            Display all details about one runtime identified by Shoot name in JSON format
  kcp runtimes --account CA4836781TID000000000123456789  Display all runtimes of a given Global Account`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(cobraCmd *cobra.Command, _ []string) error { return cmd.Run(cobraCmd) },
	}

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Shoots, "shoot", "c", nil, "Filter by Shoot cluster name. Multiple values can be provided, either separated as a comma (e.g shoot1,shoot2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.GlobalAccountIDs, "account", "g", nil, "Filter by Global Account ID. Multiple values can be provided, either separated as a comma (e.g GAID1,GAID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.SubAccountIDs, "subaccount", "s", nil, "Filter by Subaccount ID. Multiple values can be provided, either separated as a comma (e.g SAID1,SAID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.RuntimeIDs, "runtime-id", "i", nil, "Filter by Runtime ID. Multiple values can be provided, either separated as a comma (e.g ID1,ID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Regions, "region", "r", nil, "Filter by Region. Multiple values can be provided, either separated as a comma (e.g cf-eu10,cf-us10), or by specifying the option multiple times")

	return cobraCmd
}

// Run executes the runtimes command
func (cmd *RuntimeCommand) Run(cobraCmd *cobra.Command) error {
	client := runtime.NewClient(cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))

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
	switch rt.Status.Provisioning.State {
	case inProgress:
		return "provisioning"
	case failed:
		return "failed (provision)"
	}
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
	if rt.Status.UpgradingKyma != nil {
		switch rt.Status.UpgradingKyma.State {
		case inProgress:
			return "upgrading"
		case failed:
			return "failed (upgrade)"
		case succeeded:
			return "succeeded"
		}
	}

	return "succeeded"
}

func runtimeCreatedAt(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return rt.Status.CreatedAt.Format("2006/01/02 15:04:05")
}
