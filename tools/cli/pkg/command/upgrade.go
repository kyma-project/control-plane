package command

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

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
	version             string
	orchestrationParams orchestration.Parameters
}

var scheduleInputToParam = map[string]orchestration.ScheduleType{
	"":                  "",
	"immediate":         "immediate",
	"maintenancewindow": "maintenanceWindow",
}

// NewUpgradeCmd constructs the upgrade command and all subcommands under the upgrade command
func NewUpgradeCmd() *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"u"},
		Short:   "Performs upgrade operations on Kyma Runtimes.",
		Long:    "Performs upgrade operations on Kyma Runtimes.",
	}

	cobraCmd.AddCommand(NewUpgradeKymaCmd())
	return cobraCmd
}

// SetUpgradeOpts configures the upgrade specific options on the given command
func (cmd *UpgradeCommand) SetUpgradeOpts(cobraCmd *cobra.Command) {
	SetRuntimeTargetOpts(cobraCmd, &cmd.targetInputs, &cmd.targetExcludeInputs)
	cobraCmd.Flags().StringVar(&cmd.strategy, "strategy", string(orchestration.ParallelStrategy), "Orchestration strategy to use.")
	cobraCmd.Flags().IntVar(&cmd.orchestrationParams.Strategy.Parallel.Workers, "parallel-workers", 0, "Number of parallel workers to use in parallel orchestration strategy. By default the amount of workers will be auto-selected on control plane server side.")
	cobraCmd.Flags().StringVar(&cmd.schedule, "schedule", "", "Orchestration schedule to use. Possible values: \"immediate\", \"maintenancewindow\". By default the schedule will be auto-selected on control plane server side.")
	cobraCmd.Flags().StringVarP(&cmd.version, "version", "v", "", "Kyma version to use. Supports semantic (1.18.0), PR-<number> (PR-123), and <branch name>-<commit hash> (master-00e83e99) as values.")
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

	// Validate version
	// More advanced Kyma validation (via git resolution) is handled by KEB
	if err = ValidateUpgradeKymaVersionFmt(cmd.version); err != nil {
		return err
	}
	cmd.orchestrationParams.Version = cmd.version

	return nil
}

func ValidateUpgradeKymaVersionFmt(version string) error {
	switch {
	// handle semantic version
	case semver.IsValid(fmt.Sprintf("v%s", version)):
		return nil
	// handle PR-<number>
	case strings.HasPrefix(version, "PR-"):
		return nil
	// handle <branch name>-<commit hash>
	case strings.Contains(version, "-"):
		return nil
	}

	return fmt.Errorf("unsupported version format: %s", version)
}
