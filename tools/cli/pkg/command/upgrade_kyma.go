package command

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// UpgradeKymaCommand represents an execution of the kcp upgrade kyma command. Inherits fields and methods of UpgradeCommand
type UpgradeKymaCommand struct {
	UpgradeCommand
	version  string
	cobraCmd *cobra.Command
}

// NewUpgradeKymaCmd constructs a new instance of UpgradeKymaCommand and configures it in terms of a cobra.Command
func NewUpgradeKymaCmd() *cobra.Command {
	cmd := UpgradeKymaCommand{UpgradeCommand: UpgradeCommand{}}
	cobraCmd := &cobra.Command{
		Use:   "kyma --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...]",
		Short: "Upgrades or reconfigures Kyma on one or more Kyma Runtimes.",
		Long: `Upgrades or reconfigures Kyma on targets of Runtimes.
The upgrade is performed by Kyma Control Plane (KCP) within a new orchestration asynchronously. The ID of the orchestration is returned by the command upon success.
The targets of Runtimes are specified via the --target and --target-exclude options. At least one --target must be specified.
The version is specified using the --version (or -v) option. If not specified, the version is configured by Kyma Environment Broker (KEB).
Additional Kyma configurations to use for the upgrade are taken from Kyma Control Plane during the processing of the orchestration.`,
		Example: `  kcp upgrade kyma --target all --schedule maintenancewindow     Upgrade Kyma on all Runtimes in their next respective maintenance window hours.
  kcp upgrade kyma --target "account=CA.*"                       Upgrade Kyma on Runtimes of all global accounts starting with CA.
  kcp upgrade kyma --target all --target-exclude "account=CA.*"  Upgrade Kyma on Runtimes of all global accounts not starting with CA.
  kcp upgrade kyma --target "region=europe|eu|uk"                Upgrade Kyma on Runtimes whose region belongs to Europe.
  kcp upgrade kyma --target all --version "main-00e83e99"        Upgrade Kyma on Runtimes of all global accounts to the custom Kyma version (main-00e83e99).`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cmd.SetUpgradeOpts(cobraCmd)
	return cobraCmd
}

// SetUpgradeOpts configures the upgrade kyma specific options on the given command
func (cmd *UpgradeKymaCommand) SetUpgradeOpts(cobraCmd *cobra.Command) {
	cmd.UpgradeCommand.SetUpgradeOpts(cobraCmd)
	cobraCmd.Flags().StringVar(&cmd.version, "version", "", "Kyma version to use. Supports semantic (1.18.0), PR-<number> (PR-123), and <branch name>-<commit hash> (main-00e83e99) as values.")
}

// Run executes the upgrade kyma command
func (cmd *UpgradeKymaCommand) Run() error {
	cmd.log = logger.New()
	client := orchestration.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))
	ur, err := client.UpgradeKyma(cmd.orchestrationParams)
	if err != nil {
		return errors.Wrap(err, "while triggering kyma upgrade")
	}
	fmt.Println("OrchestrationID:", ur.OrchestrationID)

	if !cmd.orchestrationParams.DryRun {
		slack_title := `upgrade kyma`
		slack_err := SendSlackNotification(slack_title, cmd.cobraCmd, "OrchestrationID:"+ur.OrchestrationID)
		if slack_err != nil {
			return errors.Wrap(slack_err, "while sending notification to slack")
		}
	}
	return nil
}

// Validate checks the input parameters of the upgrade kyma command
func (cmd *UpgradeKymaCommand) Validate() error {
	err := cmd.ValidateTransformUpgradeOpts()
	if err != nil {
		return err
	}

	// Validate version
	// More advanced Kyma validation (via git resolution) is handled by KEB
	if err = ValidateUpgradeKymaVersionFmt(cmd.version); err != nil {
		return err
	}
	if cmd.orchestrationParams.Kyma == nil {
		cmd.orchestrationParams.Kyma = &orchestration.KymaParameters{Version: cmd.version}
	} else {
		cmd.orchestrationParams.Kyma.Version = cmd.version
	}

	if GlobalOpts.SlackAPIURL() == "" {
		fmt.Println("Note: Ignore sending slack notification when slackAPIURL is empty")
	}

	return nil
}

func ValidateUpgradeKymaVersionFmt(version string) error {
	switch {
	// default empty is allowed
	case version == "":
		return nil
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
