package command

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/credential"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/deprovision"
)

type DeprovisionCommand struct {
	cobraCmd        *cobra.Command
	log             logger.Logger
	shootName       string
	globalAccountID string
	subAccountID    string
	runtimeID       string
	outputPath      string
}

func NewDeprovisionCmd() *cobra.Command {
	cmd := DeprovisionCommand{}
	cobraCmd := &cobra.Command{
		Use:     "deprovision",
		Aliases: []string{"d"},
		Short:   "Deprovisions a Kyma Runtime",
		Long: `Deprovisions a Kyma Runtime.
The Runtime can be specified by one of the following:
  - Global account / Runtime ID pair with the --account and --runtime-id options
  - Shoot cluster name with the --shoot option.

By default, the deprovision file is saved to the current directory. The output file name can be specified using the --output option.`,
		Example: `  kcp deprovision -g GAID -r RUNTIMEID                    Deprovisions the SKR using Runtime ID.
  kcp deprovision -c c-178e034                            Deprovisions the SKR using a Shoot cluster name.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.outputPath, "output", "o", "", "Path to the file to save the downloaded kubeconfig to. Defaults to {CLUSTER NAME}.yaml in the current directory if not specified.")
	cobraCmd.Flags().StringVarP(&cmd.globalAccountID, "account", "g", "", "Global account ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.subAccountID, "subaccount", "s", "", "Subccount ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.shootName, "shootName", "c", "", "Shoot cluster name of the specific Kyma Runtime.")

	return cobraCmd
}

func (cmd *DeprovisionCommand) Run() error {
	// TODO: Ask the user if they are sure about the deprovision operation.
	cmd.log = logger.New()
	cred := CLICredentialManager(cmd.log)
	param := deprovision.DeprovisionParameters{ // TODO: What to choose for AuthStyle and Scope?
		ClientID:     GlobalOpts.oidcClientID,
		ClientSecret: GlobalOpts.oidcClientSecret,
		TokenURL:     GlobalOpts.oauth2URL,
		Context:      cmd.cobraCmd.Context(),
	}
	client := deprovision.NewDeprovisionClient(param)

	if cmd.runtimeID != "" {
		client.DeprovisionRuntime(cmd.runtimeID)
	} else {
		err := cmd.resolveRuntimeFromShootName(cmd.cobraCmd.Context(), cred)
		if err != nil {
			errors.Wrap(err, "while resolving runtime from shootName")
		}
		client.DeprovisionRuntime(cmd.runtimeID)
	}
	return nil
}

func (cmd *DeprovisionCommand) Validate() error {
	if cmd.globalAccountID != "" && (cmd.subAccountID != "" || cmd.runtimeID != "") || cmd.shootName != "" {
		return nil
	}
	return errors.New("at least one of the following options have to be specified: account/subaccount, account/runtime-id, shoot")

	if !promptUser(fmt.Sprintf("Runtime: '%s' will be deprovisioned. Are you sure you want to continue? ", cmd.runtimeID)) {
		return errors.New("deprovision command aborted")
	}

	return nil
}

func (cmd *DeprovisionCommand) resolveRuntimeFromShootName(ctx context.Context, cred credential.Manager) error {
	httpClient := oauth2.NewClient(ctx, cred)
	rtClient := runtime.NewClient(GlobalOpts.KEBAPIURL(), httpClient)
	params := runtime.ListParameters{}
	if cmd.shootName != "" {
		params.Shoots = []string{cmd.shootName}
	}
	if cmd.globalAccountID != "" {
		params.GlobalAccountIDs = []string{cmd.globalAccountID}
	}
	if cmd.subAccountID != "" {
		params.SubAccountIDs = []string{cmd.subAccountID}
	}

	rp, err := rtClient.ListRuntimes(params)
	if err != nil {
		return err
	}
	if rp.Count < 1 {
		return fmt.Errorf("no runtimes matched the input options")
	}
	if rp.Count > 1 {
		return fmt.Errorf("multiple runtimes (%d) matched the input options", rp.Count)
	}

	cmd.runtimeID = rp.Data[0].RuntimeID

	return nil
}

func promptUser(msg string) bool {
	fmt.Printf("%s%s", "? ", msg)
	for {
		fmt.Print("Type [y/N]: ")
		var res string
		if _, err := fmt.Scanf("%s", &res); err != nil {
			return false
		}
		switch res {
		case "yes", "y":
			return true
		case "No", "N", "no", "n":
			return false
		default:
			continue
		}
	}
}
