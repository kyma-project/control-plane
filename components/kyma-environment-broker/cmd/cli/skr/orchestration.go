package skr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/spf13/cobra"
)

// OrchestrationCommand represents an execution of the skr orchestrations command
type OrchestrationCommand struct {
	log       logger.Logger
	output    string
	state     string
	operation string
}

// NewOrchestrationCmd constructs a new instance of OrchestrationCommand and configures it in terms of a cobra.Command
func NewOrchestrationCmd(log logger.Logger) *cobra.Command {
	cmd := OrchestrationCommand{log: log}
	cobraCmd := &cobra.Command{
		Use:     "orchestrations [id]",
		Aliases: []string{"orchestration", "o"},
		Short:   "Display Kyma control plane orchestrations",
		Long: `Display Kyma control plane orchestrations and their primary attributes, such as identifiers, type, state, parameters, runtime operations.
The commands has two modes:
  1. Without specifying an orcherstration id as an argument, it will list all orchestrations, or orcherstrations matching the --state if supplied.
  2. When specifying an orcherstration id as an argument, it will display details about the specific orchestration.
     If the optional --operation is given, it will display details of the specified runtime operation within the orchestration.`,
		Example: `  skr orchestrations --state inprogress                                   Display all orchestrations which are in progress
  skr orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00                  Display details about a specific orchestration
  skr orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 --operation OID  Display details of the specified runtime operation within the orchestration`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error { return cmd.Validate(args) },
		RunE:    func(_ *cobra.Command, args []string) error { return cmd.Run(args) },
	}

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringVarP(&cmd.state, "state", "s", "", fmt.Sprintf("Filter output by state. Possible values: %s", strings.Join(allOrchestrationStates(), ", ")))
	cobraCmd.Flags().StringVar(&cmd.operation, "operation", "", "Display details of the specified runtime operation when a given orchestration is selected")
	return cobraCmd
}

func orchestrationToCLIState(state string) string {
	return strings.ReplaceAll(state, " ", "")
}

func allOrchestrationStates() []string {
	var states = []string{}
	for _, state := range []string{internal.Pending, internal.InProgress, internal.Succeeded, internal.Failed} {
		states = append(states, orchestrationToCLIState(state))
	}

	return states
}

func validateOrchestrationState(inputState string, args []string) error {
	if inputState == "" {
		return nil
	} else if len(args) > 0 {
		return errors.New("--state should not be used together with orchestration argument")
	}
	for _, state := range allOrchestrationStates() {
		if state == inputState {
			return nil
		}
	}

	return fmt.Errorf("invalid value for state: %s", inputState)
}

// Run executes the orchestrations command
func (cmd *OrchestrationCommand) Run(args []string) error {
	fmt.Println("Not implemented yet.")
	return nil
}

// Validate checks the input parameters of the orchestrations command
func (cmd *OrchestrationCommand) Validate(args []string) error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}

	err = validateOrchestrationState(cmd.state, args)
	if err != nil {
		return err
	}

	if cmd.operation != "" && len(args) == 0 {
		return errors.New("--operation shoiuld only be used when orchestration id is given as an argument")
	}

	return nil
}
