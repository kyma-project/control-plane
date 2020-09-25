package skr

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RuntimeCommand represents an execution of the skr runtimes command
type RuntimeCommand struct {
	output           string
	shoots           []string
	globalAccountIDs []string
	subAccountIDs    []string
	runtimeIDs       []string
	instanceIDs      []string
	regions          []string
}

// NewRuntimeCmd constructs a new instance of RuntimeCommand and configures it in terms of a cobra.Command
func NewRuntimeCmd() *cobra.Command {
	cmd := RuntimeCommand{}
	cobraCmd := &cobra.Command{
		Use:     "runtimes",
		Aliases: []string{"runtime", "rt"},
		Short:   "Display Kyma runtimes",
		Long: `Display Kyma runtimes and their primary attributes, such as identifiers, region, states, etc.
The command supports filtering runtimes based on various attributes, see the list of options below.`,
		Example: `  skr runtimes                                           Display table overview about all runtimes
  skr rt -c c-178e034 -o json                            Display all details about one runtime identified by Shoot name in JSON format
  skr runtimes --account CA4836781TID000000000123456789  Display all runtimes of a given Global Account`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.shoots, "shoot", "c", nil, "Filter by Shoot cluster name. Multiple values can be provided, either separated as a comma (e.g shoot1,shoot2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.globalAccountIDs, "account", "g", nil, "Filter by Global Account ID. Multiple values can be provided, either separated as a comma (e.g GAID1,GAID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.subAccountIDs, "subaccount", "s", nil, "Filter by Subaccount ID. Multiple values can be provided, either separated as a comma (e.g SAID1,SAID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.runtimeIDs, "runtime-id", "i", nil, "Filter by Runtime ID. Multiple values can be provided, either separated as a comma (e.g ID1,ID2), or by specifying the option multiple times")
	cobraCmd.Flags().StringSliceVarP(&cmd.regions, "region", "r", nil, "Filter by Region. Multiple values can be provided, either separated as a comma (e.g cf-eu10,cf-us10), or by specifying the option multiple times")

	return cobraCmd
}

// Run executes the runtimes command
func (cmd *RuntimeCommand) Run() error {
	fmt.Println("Not implemented yet.")
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
