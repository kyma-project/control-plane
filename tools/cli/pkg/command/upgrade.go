package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
)

// UpgradeCommand is the base type of all subcommands under the upgrade command. The type holds common attributes and methods inherited by all subcommands
type UpgradeCommand struct {
	log                 logger.Logger
	targetInputs        []string
	targetExcludeInputs []string
	strategy            string
	schedule            string
	orchestrationParams orchestration.Parameters
}

var scheduleInputToParam = map[string]orchestration.ScheduleType{
	"":                  "",
	"immediate":         "immediate",
	"maintenancewindow": "maintenanceWindow",
}

// NewUpgradeCmd constructs the upgrade command and all subcommands under the upgrade command
func NewUpgradeCmd(log logger.Logger) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"u"},
		Short:   "Performs upgrade operations on Kyma Runtimes.",
		Long:    "Performs upgrade operations on Kyma Runtimes.",
	}

	cobraCmd.AddCommand(NewUpgradeKymaCmd(log))
	return cobraCmd
}

// SetUpgradeOpts configures the upgrade specific options on the given command
func (cmd *UpgradeCommand) SetUpgradeOpts(cobraCmd *cobra.Command) {
	SetRuntimeTargetOpts(cobraCmd, &cmd.targetInputs, &cmd.targetExcludeInputs)
	cobraCmd.Flags().StringVar(&cmd.strategy, "strategy", string(orchestration.ParallelStrategy), "Orchestration strategy to use.")
	cobraCmd.Flags().IntVar(&cmd.orchestrationParams.Strategy.Parallel.Workers, "parallel-workers", 0, "Number of parallel workers to use in parallel orchestration strategy. By default the amount of workers will be auto-selected on control plane server side.")
	cobraCmd.Flags().StringVar(&cmd.schedule, "schedule", "", "Orchestration schedule to use. Possible values: \"immediate\", \"maintenancewindow\". By default the schedule will be auto-selected on control plane server side.")
	cobraCmd.Flags().BoolVar(&cmd.orchestrationParams.DryRun, "dry-run", false, "Perform the orchestration without executing the actual upgrage operations for the Runtimes. The details can be obtained using the \"kcp orchestrations\" command.")
}

// ValidateTransformUpgradeOpts checks in the input upgrade options, and transforms them for internal usage
func (cmd *UpgradeCommand) ValidateTransformUpgradeOpts() error {
	err := ValidateTransformRuntimeTargetOpts(cmd.targetInputs, cmd.targetExcludeInputs, &cmd.orchestrationParams.Targets)
	if err != nil {
		return err
	}

	// Validate schedule
	if scheduleParam, ok := scheduleInputToParam[cmd.schedule]; ok {
		cmd.orchestrationParams.Strategy.Schedule = scheduleParam
	} else {
		return fmt.Errorf("invalid value for schedule: %s. Check kcp upgrade --help for more information", cmd.schedule)
	}

	// Validate strategy type
	switch cmd.strategy {
	case string(orchestration.ParallelStrategy):
		cmd.orchestrationParams.Strategy.Type = orchestration.StrategyType(cmd.strategy)
	default:
		return fmt.Errorf("invalid value for strategy: %s", cmd.strategy)
	}

	return nil
}
