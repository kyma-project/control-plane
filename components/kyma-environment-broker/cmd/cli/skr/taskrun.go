package skr

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/spf13/cobra"
)

// TaskRunCommand represents an execution of the skr taskrun command
type TaskRunCommand struct {
	logger              logger.Logger
	targetInputs        []string
	targetExcludeInputs []string
	parallelism         int
	targets             internal.TargetSpec
	kubeconfigDir       string
	keepKubeconfigs     bool
}

// NewTaskRunCmd constructs a new instance of TaskRunCommand and configures it in terms of a cobra.Command
func NewTaskRunCmd(logger logger.Logger) *cobra.Command {
	cmd := TaskRunCommand{logger: logger}
	cobraCmd := &cobra.Command{
		Use:     "taskrun --target <TARGET SPEC> ... [--target-exclude <TARGET SPEC> ...] COMMAND [ARGS ...]",
		Aliases: []string{"task", "t"},
		Short:   "Run generic tasks on one or more Kyma runtimes",
		Long: `Runs a command (which can be a script or a program with arbitrary arguments) on targets of Kyma runtimes.
The specified command will be executed locally in parallel in separate subprocesses for each runtime, where the number of parallel executions are controlled by the --parallelism option.

For each subprocess, the following runtime specific data are passed as environment variables:
  KUBECONFIG       : Path to the kubeconfig file for the specific SKR
  GLOBALACCOUNT_ID : Global Account ID of the SKR
  SUBACCOUNT_ID    : Subaccount ID of the SKR
  RUNTIME_NAME     : Shoot cluster name
  RUNTIME_ID       : Runtime ID of the SKR

The exit status is zero (0) if all subprocesses exit successfully with zero status code. If one or more subprocesses exit with non-zero status, the command will also exit with non-zero status.`,
		Example: `  skr taskrun --target all kubectl patch deployment valid-deployment -p '{"spec":{"template":{"spec":{"containers":[{"name":"kubernetes-serve-hostname","image":"new image"}]}}}}'
    Execute a kubectl patch operation for all runtimes
  skr taskrun --target account=CA4836781TID000000000123456789 /usr/local/bin/awesome-script.sh
    Run a maintenance script for all runtimes of a given Global Account
  skr taskrun --target all helm upgrade -i -n kyma-system my-kyma-addon --values overrides.yaml
    Deploy or a helm chart on all runtimes`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	SetRuntimeTargetOpts(cobraCmd, &cmd.targetInputs, &cmd.targetExcludeInputs)
	cobraCmd.Flags().IntVarP(&cmd.parallelism, "parallelism", "p", 8, "Number of parallel commands to execute")
	cobraCmd.Flags().StringVar(&cmd.kubeconfigDir, "kubeconfig-dir", "", "Directory to download runtime kubeconfigs to. By default it is a random-generated directory in the OS specific default temporary directory (e.g. /tmp in Linux)")
	cobraCmd.Flags().BoolVar(&cmd.keepKubeconfigs, "keep", false, "Keep downloaded kubeconfigs after execution for caching purpose")
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
