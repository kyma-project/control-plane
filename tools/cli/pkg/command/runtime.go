package command

import (
	"fmt"
	"strings"

	"golang.org/x/oauth2"

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
	states   []string
	opDetail bool
	display  Display
}

type Display struct {
	SubscriptionGlobalAccountID bool
}

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
                                                         Display all Runtimes with specific custom fields.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Shoots, "shoot", "c", nil, "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.GlobalAccountIDs, "account", "g", nil, "Filter by global account ID. You can provide multiple values, either separated by a comma (e.g. GAID1,GAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.SubAccountIDs, "subaccount", "s", nil, "Filter by subaccount ID. You can provide multiple values, either separated by a comma (e.g. SAID1,SAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().BoolVar(&cmd.display.SubscriptionGlobalAccountID, "subscription-global-account-id", false, "Display Subscription Global Account ID.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.InstanceIDs, "instance-id", "i", nil, "Filter by instance ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.RuntimeIDs, "runtime-id", "r", nil, "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Regions, "region", "R", nil, "Filter by provider region. You can provide multiple values, either separated by a comma (e.g. westeurope,northeurope), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Plans, "plan", "p", nil, "Filter by service plan name. You can provide multiple values, either separated by a comma (e.g. azure,trial), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.states, "state", "S", nil, "Filter by Runtime state. The possible values are: succeeded, failed, error, provisioning, deprovisioning, upgrading, suspended, all. Suspended Runtimes are filtered out unless the \"all\" or \"suspended\" values are provided. You can provide multiple values, either separated by a comma (e.g. succeeded,failed), or by specifying the option multiple times.")
	cobraCmd.Flags().BoolVar(&cmd.opDetail, "ops", false, "Get all operations for the runtimes instead of just querying the last operation.")
	cobraCmd.Flags().BoolVar(&cmd.params.KymaConfig, "kyma-config", false, "Get all Kyma configuration details for the selected runtimes.")
	cobraCmd.Flags().BoolVar(&cmd.params.ClusterConfig, "cluster-config", false, "Get all cluster configuration details for the selected runtimes.")
	cobraCmd.Flags().BoolVar(&cmd.params.Expired, "expired", false, "Lists only expired runtimes")

	return cobraCmd
}

// Run executes the runtimes command
func (cmd *RuntimeCommand) Run() error {
	cmd.log = logger.New()
	httpClient := oauth2.NewClient(cmd.cobraCmd.Context(), CLICredentialManager(cmd.log))
	client := runtime.NewClient(GlobalOpts.KEBAPIURL(), httpClient)

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

//Test
// Validate checks the input parameters of the runtimes command
func (cmd *RuntimeCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}

	// Validate and transform states
	for _, s := range cmd.states {
		val := runtime.State(s)
		switch val {
		case runtime.StateSucceeded, runtime.StateFailed, runtime.StateError, runtime.StateProvisioning, runtime.StateDeprovisioning, runtime.StateUpgrading, runtime.StateSuspended, runtime.AllState:
			cmd.params.States = append(cmd.params.States, val)
		default:
			return fmt.Errorf("invalid value for state: %s", s)
		}
	}

	cmd.params.OperationDetail = runtime.LastOperation
	if cmd.opDetail {
		cmd.params.OperationDetail = runtime.AllOperation
	}

	return nil
}

func (cmd *RuntimeCommand) printRuntimes(runtimes runtime.RuntimesPage) error {
	switch {
	case cmd.output == tableOutput:
		if cmd.display.SubscriptionGlobalAccountID {
			tableColumns = append(tableColumns[:1+1], tableColumns[1:]...)
			tableColumns[1] = printer.Column{
				Header:    "Subscription Global Account ID",
				FieldSpec: "{.subscriptionGlobalAccountID}",
			}
		}

		if cmd.opDetail {
			tableColumns = append(tableColumns, printer.Column{
				Header:    "KYMA VERSION",
				FieldSpec: "{.KymaVersion}",
			})
		}
		tp, err := printer.NewTablePrinter(tableColumns, false)
		if err != nil {
			return err
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
		return ccp.PrintObj(runtimes.Data)
	}
	return nil
}

func runtimeStatus(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	state := rt.Status.State
	switch state {
	case runtime.StateError, runtime.StateFailed:
		op := rt.LastOperation()
		state = runtime.State(fmt.Sprintf("%s (%s)", state, op.Type))
	}

	return string(state)
}

func runtimeCreatedAt(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return rt.Status.CreatedAt.Format("2006/01/02 15:04:05")
}
