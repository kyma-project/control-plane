package command

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/spf13/cobra"
)

// UpgradeKymaCommand represents an execution of the kcp upgrade kyma command. Inherits fields and methods of UpgradeCommand
type UpgradeKymaCommand struct {
	UpgradeCommand
}

// NewUpgradeKymaCmd constructs a new instance of UpgradeKymaCommand and configures it in terms of a cobra.Command
func NewUpgradeKymaCmd(log logger.Logger) *cobra.Command {
	cmd := UpgradeKymaCommand{
		UpgradeCommand: UpgradeCommand{
			log: log,
		},
	}
	cobraCmd := &cobra.Command{
		Use:   "kyma --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...]",
		Short: "Upgrades or reconfigures Kyma on one or more Kyma Runtimes.",
		Long: `Upgrades or reconfigures Kyma on targets of Runtimes.
The upgrade is performed by Kyma Control Plane (KCP) within a new orchestration asynchronously. The ID of the orchestration is returned by the command upon success.
The targets of Runtimes are specified via the --target and --target-exclude options. At least one --target must be specified.
The Kyma version and configurations to use for the upgrade are taken from Kyma Control Plane during the processing of the orchestration.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		Example: `  kcp upgrade kyma --target all --schedule maintenancewindow     Upgrade Kyma on all Runtimes in their next respective maintenance window hours.
  kcp upgrade kyma --target "account=CA.*"                       Upgrade Kyma on Runtimes of all global accounts starting with CA.
  kcp upgrade kyma --target all --target-exclude "account=CA.*"  Upgrade Kyma on Runtimes of all global accounts not starting with CA.
  kcp upgrade kyma --target "region=europe|eu|uk"                Upgrade Kyma on Runtimes whose region belongs to Europe.`,
		RunE: func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cmd.SetUpgradeOpts(cobraCmd)
	return cobraCmd
}

// Run executes the upgrade kyma command
func (cmd *UpgradeKymaCommand) Run() error {
	fmt.Println("Not implemented yet.")
	return nil
}

// Validate checks the input parameters of the upgrade kyma command
func (cmd *UpgradeKymaCommand) Validate() error {
	err := cmd.ValidateTransformUpgradeOpts()
	if err != nil {
		return err
	}
	return nil
}
