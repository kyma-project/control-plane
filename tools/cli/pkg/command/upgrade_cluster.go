package command

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type UpgradeClusterCommand struct {
	UpgradeCommand
	version  string
	cobraCmd *cobra.Command
}

func NewUpgradeClusterCommand() *cobra.Command {
	cmd := UpgradeClusterCommand{UpgradeCommand: UpgradeCommand{}}
	cobraCmd := &cobra.Command{
		Use:   "cluster --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...]",
		Short: "Upgrades Kubernetes cluster on one or more Kyma Runtimes.",
		Long: `Upgrade Kubernetes cluster on targets of Runtimes.
The upgrade is performed by Kyma Control Plane (KCP) within a new orchestration asynchronously. The ID of the orchestration is returned by the command upon success.
The targets of Runtimes are specified via the --target and --target-exclude options. At least one --target must be specified.
The version is specified using the --version (or -v) option.
Additional Kyma configurations to use for the upgrade are taken from Kyma Control Plane during the processing of the orchestration.`,
		Example: `kcp upgrade cluster --target all --schedule maintenancewindow    Upgrade Kubernetes cluster on Runtime in their next respective maintenance window hours.
		kcp upgrade cluster --target "account=CA.*"                       Upgrade Kubernetes cluster on Runtimes of all global accounts starting with CA.
		kcp upgrade cluster --target all --target-exclude "account=CA.*"  Upgrade Kubernetes cluster on Runtimes of all global accounts not starting with CA.
		kcp upgrade cluster --target "region=europe|eu|uk"                Upgrade Kubernetes cluster on Runtimes whose region belongs to Europe.
		kcp upgrade cluster --target all --version "v1.20.0"      Upgrade Kubernetes cluster on Runtimes of all global accounts to the custom Kubernetes cluster version (1.20.0).`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cmd.cobraCmd = cobraCmd

	cmd.SetUpgradeClusterOpts(cobraCmd)
	return cobraCmd
}

// SetUpgradeOpts configures the upgrade kyma specific options on the given command
func (cmd *UpgradeClusterCommand) SetUpgradeClusterOpts(cobraCmd *cobra.Command) {
	cmd.UpgradeCommand.SetUpgradeOpts(cobraCmd)
	cobraCmd.Flags().StringVar(&cmd.version, "version", "", "Kubernetes version to use. like v1.21.0 as values.")
}

func (cmd *UpgradeClusterCommand) Validate() error {
	err := cmd.ValidateTransformUpgradeOpts()
	if err != nil {
		return err
	}

	cmd.orchestrationParams.Version = cmd.version

	return nil
}

// Run executes the upgrade cluster command
func (cmd *UpgradeClusterCommand) Run() error {
	cmd.log = logger.New()
	client := orchestration.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))
	ur, err := client.UpgradeCluster(cmd.orchestrationParams)
	if err != nil {
		return errors.Wrap(err, "while triggering kyma upgrade")
	}
	fmt.Println("OrchestrationID:", ur.OrchestrationID)
	return nil
}
