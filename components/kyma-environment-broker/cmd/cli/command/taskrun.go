package command

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/spf13/cobra"
)

// TaskRunCommand represents an execution of the kcp taskrun command
type TaskRunCommand struct {
	log                 logger.Logger
	targetInputs        []string
	targetExcludeInputs []string
	parallelism         int
	targets             internal.TargetSpec
	kubeconfigDir       string
	keepKubeconfigs     bool
}

// NewTaskRunCmd constructs a new instance of TaskRunCommand and configures it in terms of a cobra.Command
func NewTaskRunCmd(log logger.Logger) *cobra.Command {
	cmd := TaskRunCommand{log: log}
	cobraCmd := &cobra.Command{
		Use:     "taskrun --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...] COMMAND [ARGS ...]",
		Aliases: []string{"task", "t"},
		Short:   "Runs generic tasks on one or more Kyma Runtimes.",
		Long: `Runs a command, which can be a script or a program with arbitrary arguments, on targets of Kyma Runtimes.
The specified command is executed locally. It is executed in separate subprocesses for each Runtime in parallel, where the number of parallel executions is controlled by the --parallelism option.

For each subprocess, the following Runtime-specific data are passed as environment variables:
  - KUBECONFIG       : Path to the kubeconfig file for the specific Runtime
  - GLOBALACCOUNT_ID : Global account ID of the Runtime
  - SUBACCOUNT_ID    : Subaccount ID of the Runtime
  - RUNTIME_NAME     : Shoot cluster name
  - RUNTIME_ID       : Runtime ID of the Runtime

	If all subprocesses finish successfully with the zero status code, the exit status is zero (0). If one or more subprocesses exit with a non-zero status, the command will also exit with a non-zero status.`,
		Example: `  kcp taskrun --target all kubectl patch deployment valid-deployment -p '{"metadata":{"labels":{"my-label": "my-value"}}}'
    Execute a kubectl patch operation for all Runtimes.
  kcp taskrun --target account=CA4836781TID000000000123456789 /usr/local/bin/awesome-script.sh
    Run a maintenance script for all Runtimes of a given global account.
  kcp taskrun --target all helm upgrade -i -n kyma-system my-kyma-addon --values overrides.yaml
    Deploy a helm chart on all Runtimes.`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetRuntimeTargetOpts(cobraCmd, &cmd.targetInputs, &cmd.targetExcludeInputs)
	cobraCmd.Flags().IntVarP(&cmd.parallelism, "parallelism", "p", 8, "Number of parallel commands to execute.")
	cobraCmd.Flags().StringVar(&cmd.kubeconfigDir, "kubeconfig-dir", "", "Directory to download Runtime kubeconfig files to. By default, it is a random-generated directory in the OS-specific default temporary directory (e.g. /tmp in Linux).")
	cobraCmd.Flags().BoolVar(&cmd.keepKubeconfigs, "keep", false, "Option that allows you to keep downloaded kubeconfig files after execution for caching purposes.")
	return cobraCmd
}

// Run executes the taskrun command
func (cmd *TaskRunCommand) Run() error {
	fmt.Println("Not implemented yet.")
	return nil
}

// Validate checks the input parameters of the taskrun command
func (cmd *TaskRunCommand) Validate() error {
	err := ValidateTransformRuntimeTargetOpts(cmd.targetInputs, cmd.targetExcludeInputs, &cmd.targets)
	if err != nil {
		return err
	}
	// TODO: check if cmd.kubeconfigDir dir exists if input was given
	return nil
}
