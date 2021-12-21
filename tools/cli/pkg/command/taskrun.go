package command

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/client"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/strategies"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/credential"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TaskRunCommand represents an execution of the kcp taskrun command
type TaskRunCommand struct {
	cobraCmd            *cobra.Command
	log                 logger.Logger
	cred                credential.Manager
	targetInputs        []string
	targetExcludeInputs []string
	parallelism         int
	targets             orchestration.TargetSpec
	kubeconfigDir       string
	kubeconfingDirTemp  bool
	keepKubeconfigs     bool
	noKubeconfig        bool
	noPrefixOutput      bool
	taskCommand         *exec.Cmd
	shell               string
}

// RuntimeLister implements the interface to obtains runtimes info from KEB for resolver
type RuntimeLister struct {
	client runtime.Client
}

// RuntimeTask is the runtime operation executed by RuntimeTaskMakager via strategy.
type RuntimeTask struct {
	operation orchestration.RuntimeOperation
	result    error
}

// RuntimeTaskMakager implements Executor interface needed by strategy to execute the runtime task operations.
type RuntimeTaskMakager struct {
	cmd              *TaskRunCommand
	tasks            map[string]*RuntimeTask
	kubeconfigClient client.Client
}

// TaskRunError represents failure in task execution for one or more runtimes
type TaskRunError struct {
	failed int
	total  int
}

func (e *TaskRunError) Error() string {
	return fmt.Sprintf("%d/%d execution(s) failed", e.failed, e.total)
}

// NewTaskRunCmd constructs a new instance of TaskRunCommand and configures it in terms of a cobra.Command
func NewTaskRunCmd() *cobra.Command {
	cmd := TaskRunCommand{}
	cobraCmd := &cobra.Command{
		Use:     "taskrun --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...] -- COMMAND [ARGS ...]",
		Aliases: []string{"task", "t"},
		Short:   "Runs generic tasks on one or more Kyma Runtimes.",
		Long: `Runs a command, which can be a script or a program with arbitrary arguments, on targets of Kyma Runtimes.
The specified command is executed locally. It is executed in separate subprocesses for each Runtime in parallel, where the number of parallel executions is controlled by the --parallelism option.

For each subprocess, the following Runtime-specific data are passed as environment variables:
  - KUBECONFIG       : Path to the kubeconfig file for the specific Runtime, unless --no-kubeconfig option is passed
  - GLOBALACCOUNT_ID : Global account ID of the Runtime
  - SUBACCOUNT_ID    : Subaccount ID of the Runtime
  - RUNTIME_NAME     : Shoot cluster name
  - RUNTIME_ID       : Runtime ID of the Runtime
  - INSTANCE_ID      : Instance ID of the Runtime

  If all subprocesses finish successfully with the zero status code, the exit status is zero (0). If one or more subprocesses exit with a non-zero status, the command will also exit with a non-zero status.`,
		Example: `  kcp taskrun --target all -- kubectl patch deployment valid-deployment -p '{"metadata":{"labels":{"my-label": "my-value"}}}'
    Execute a kubectl patch operation for all Runtimes.
  kcp taskrun --target account=CA4836781TID000000000123456789 /usr/local/bin/awesome-script.sh
    Run a maintenance script for all Runtimes of a given global account.
  kcp taskrun --target all -- helm upgrade -i -n kyma-system my-kyma-addon --values overrides.yaml
    Deploy a Helm chart on all Runtimes.
  kcp taskrun -t all -s "/bin/bash -i -c" -- kc get ns
    Run an alias command (kc for kubectl) defined in user's .bashrc invocation script`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error { return cmd.Validate(args) },
		RunE:    func(_ *cobra.Command, args []string) error { return cmd.Run(args) },
	}
	cmd.cobraCmd = cobraCmd

	SetRuntimeTargetOpts(cobraCmd, &cmd.targetInputs, &cmd.targetExcludeInputs)
	cobraCmd.Flags().IntVarP(&cmd.parallelism, "parallelism", "p", 4, "Number of parallel commands to execute.")
	cobraCmd.Flags().StringVarP(&cmd.kubeconfigDir, "kubeconfig-dir", "k", "", "Directory to download Runtime kubeconfig files to. By default, it is a random-generated directory in the OS-specific default temporary directory (e.g. /tmp in Linux).")
	cobraCmd.Flags().BoolVar(&cmd.keepKubeconfigs, "keep-kubeconfig", false, "Option that allows you to keep downloaded kubeconfig files after execution for caching purposes.")
	cobraCmd.Flags().BoolVar(&cmd.noKubeconfig, "no-kubeconfig", false, "Option that turns off the downloading and exposure of the kubeconfig file for each Runtime.")
	cobraCmd.Flags().BoolVar(&cmd.noPrefixOutput, "no-prefix-output", false, "Option that omits the prefixing of each output line with the Runtime name. By default, all output lines are prepended for better traceability.")
	cobraCmd.Flags().StringP("shell", "s", "", "Invoke the task command using the given shell and it's options. Useful when the task command uses alias(es) defined in the shell's invocation scripts. Can also be set in the KCP configuration file or with the KCP_SHELL environment variable.")
	viper.BindPFlag("shell", cobraCmd.Flags().Lookup("shell"))
	return cobraCmd
}

// Run executes the taskrun command
func (cmd *TaskRunCommand) Run(args []string) error {
	cmd.log = logger.New()
	cmd.cred = CLICredentialManager(cmd.log)
	defer cmd.cleanupTempKubeConfigDir()

	operations, err := cmd.resolveOperations()
	if err != nil {
		return err
	}

	mgr := NewRuntimeTaskMakager(cmd, operations)
	strategy := strategies.NewParallelOrchestrationStrategy(mgr, cmd.log, 0)
	execID, err := strategy.Execute(operations, orchestration.StrategySpec{
		Type:     orchestration.ParallelStrategy,
		Schedule: orchestration.Immediate,
		Parallel: orchestration.ParallelStrategySpec{Workers: cmd.parallelism},
	})
	if err != nil {
		return errors.Wrap(err, "while executing task")
	}
	strategy.Wait(execID)

	return mgr.exitStatus()
}

// Validate checks the input parameters of the taskrun command
func (cmd *TaskRunCommand) Validate(args []string) error {
	// Validate kubeconfig-api-url global option
	if !cmd.noKubeconfig && GlobalOpts.KubeconfigAPIURL() == "" {
		return fmt.Errorf("missing required %s option", GlobalOpts.kubeconfigAPIURL)
	}

	// Validate gardener-kubeconfig global option
	if GlobalOpts.GardenerKubeconfig() == "" || GlobalOpts.GardenerNamespace() == "" {
		return fmt.Errorf("missing required %s/%s options", GlobalOpts.gardenerKubeconfig, GlobalOpts.gardenerNamespace)
	}

	// Validate target options
	err := ValidateTransformRuntimeTargetOpts(cmd.targetInputs, cmd.targetExcludeInputs, &cmd.targets)
	if err != nil {
		return err
	}

	// Validate kubeconfig directory
	if cmd.kubeconfigDir != "" {
		fi, err := os.Stat(cmd.kubeconfigDir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("%s: not a directory", cmd.kubeconfigDir)
		}
	} else if !cmd.noKubeconfig {
		cmd.kubeconfigDir, err = ioutil.TempDir("", "kubeconfig-")
		if err != nil {
			return errors.Wrap(err, "while creating temporary kubeconfig directory")
		}
		cmd.kubeconfingDirTemp = true
	}

	// Validate task command and shell wrapper
	// Construct task command object
	cmd.shell = viper.GetString("shell")
	if cmd.shell != "" {
		splitSh := strings.Split(cmd.shell, " ")
		if _, err := exec.LookPath(splitSh[0]); err != nil {
			return err
		}
		allArgs := append(splitSh[1:], strings.Join(args, " "))
		cmd.taskCommand = exec.CommandContext(cmd.cobraCmd.Context(), splitSh[0], allArgs...)
	} else {
		if _, err := exec.LookPath(args[0]); err != nil {
			return err
		}
		cmd.taskCommand = exec.CommandContext(cmd.cobraCmd.Context(), args[0], args[1:]...)
	}
	return nil
}

func (cmd *TaskRunCommand) resolveOperations() ([]orchestration.RuntimeOperation, error) {
	gardenCfg, err := gardener.NewGardenerClusterConfig(GlobalOpts.GardenerKubeconfig())
	if err != nil {
		return nil, errors.Wrap(err, "while getting Gardener kubeconfig")
	}
	gardenClient, err := gardener.NewClient(gardenCfg)
	if err != nil {
		return nil, errors.Wrap(err, "while getting Gardener client")
	}

	httpClient := oauth2.NewClient(cmd.cobraCmd.Context(), cmd.cred)
	lister := NewRuntimeLister(runtime.NewClient(GlobalOpts.KEBAPIURL(), httpClient))
	resolver := orchestration.NewGardenerRuntimeResolver(gardenClient, GlobalOpts.GardenerNamespace(), lister, cmd.log)
	runtimes, err := resolver.Resolve(cmd.targets)
	if err != nil {
		return nil, errors.Wrap(err, "while resolving targets")
	}

	cmd.log.Infof("Number of resolved runtimes: %d\n", len(runtimes))
	operations := make([]orchestration.RuntimeOperation, 0, len(runtimes))
	for _, rt := range runtimes {
		operations = append(operations, orchestration.RuntimeOperation{
			Runtime: rt,
			ID:      randomString(16),
		})
	}

	return operations, nil
}

func (cmd *TaskRunCommand) cleanupTempKubeConfigDir() error {
	var err error = nil
	if cmd.kubeconfingDirTemp {
		err = os.RemoveAll(cmd.kubeconfigDir)
	}

	return err
}

// NewRuntimeLister constructs a RuntimeLister with the given runtime.Client
func NewRuntimeLister(client runtime.Client) *RuntimeLister {
	return &RuntimeLister{client: client}
}

// ListAllRuntimes fetches all runtimes from KEB using the runtime client
func (rl RuntimeLister) ListAllRuntimes() ([]runtime.RuntimeDTO, error) {
	res, err := rl.client.ListRuntimes(runtime.ListParameters{})
	if err != nil {
		return nil, errors.Wrap(err, "while querying runtimes")
	}

	return res.Data, nil
}

// NewRuntimeTaskMakager constructs a new RuntimeTaskMakager for the given runtime operations
func NewRuntimeTaskMakager(cmd *TaskRunCommand, operations []orchestration.RuntimeOperation) *RuntimeTaskMakager {
	mgr := &RuntimeTaskMakager{
		cmd:              cmd,
		tasks:            make(map[string]*RuntimeTask, len(operations)),
		kubeconfigClient: client.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KubeconfigAPIURL(), cmd.cred),
	}
	for _, op := range operations {
		mgr.tasks[op.ID] = &RuntimeTask{
			operation: op,
		}
	}

	return mgr
}

// Execute runs the task on the runtime identified by the operationID
func (mgr *RuntimeTaskMakager) Execute(operationID string) (time.Duration, error) {
	task := mgr.tasks[operationID]
	log := mgr.cmd.log.WithField("shoot", task.operation.ShootName)

	kubeconfigPath, err := mgr.getKubeconfig(task)
	if err != nil {
		log.Errorf("Error: while getting kubeconfig: %s\n", err.Error())
		task.result = err
		return 0, err
	}

	command := *mgr.cmd.taskCommand

	// Prepare environment variables
	command.Env = os.Environ()
	command.Env = append(command.Env,
		fmt.Sprintf("GLOBALACCOUNT_ID=%s", task.operation.GlobalAccountID),
		fmt.Sprintf("SUBACCOUNT_ID=%s", task.operation.SubAccountID),
		fmt.Sprintf("RUNTIME_ID=%s", task.operation.RuntimeID),
		fmt.Sprintf("RUNTIME_NAME=%s", task.operation.ShootName),
		fmt.Sprintf("INSTANCE_ID=%s", task.operation.InstanceID),
	)
	if kubeconfigPath != "" {
		command.Env = append(command.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))
	}

	// Prepare stdout and stderr of the command
	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Errorf("Error: while creating stdout: %s\n", err.Error())
		task.result = err
		return 0, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		log.Errorf("Error: while creating stderr: %s\n", err.Error())
		task.result = err
		return 0, err
	}

	// Prepare echoer stdout / stderr writers
	echoerWg := sync.WaitGroup{}
	echoer := func(src io.Reader, dst io.Writer) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			if !mgr.cmd.noPrefixOutput {
				fmt.Fprintf(dst, "%s ", task.operation.ShootName)
			}
			fmt.Fprintln(dst, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Errorf("Error: while reading from child process: %s\n", err)
		}
		echoerWg.Done()
	}
	echoerWg.Add(2)
	go echoer(stdout, os.Stdout)
	go echoer(stderr, os.Stderr)

	// Start execution of the command
	err = command.Start()
	if err != nil {
		log.Errorf("Error: command started with error: %s\n", err.Error())
	}
	// Wait for the command subprocess to finish
	echoerWg.Wait()
	err = command.Wait()
	if err != nil {
		log.Errorf("Error: command exited with error: %s\n", err.Error())
	}
	task.result = err

	return 0, err
}

func (mgr *RuntimeTaskMakager) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}

func (mgr *RuntimeTaskMakager) getKubeconfig(task *RuntimeTask) (string, error) {
	path := ""
	if !mgr.cmd.noKubeconfig {
		path = fmt.Sprintf("%s/%s.yaml", mgr.cmd.kubeconfigDir, task.operation.ShootName)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			kubeconfig, err := mgr.kubeconfigClient.GetKubeConfig(task.operation.GlobalAccountID, task.operation.RuntimeID)
			if err != nil {
				return path, err
			}

			err = ioutil.WriteFile(path, []byte(kubeconfig), 0600)
			if err != nil {
				return path, err
			}
		}
	}

	return path, nil
}

func (mgr *RuntimeTaskMakager) exitStatus() error {
	e := &TaskRunError{total: len(mgr.tasks)}
	for _, task := range mgr.tasks {
		if task.result != nil {
			e.failed++
		}
	}
	if e.failed != 0 {
		return e
	}

	return nil
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
