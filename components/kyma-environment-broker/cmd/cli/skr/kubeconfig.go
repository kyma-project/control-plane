package skr

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
)

// KubeconfigCommand represents an execution of the skr kubeconfig command
type KubeconfigCommand struct {
	logger          logger.Logger
	shoot           string
	globalAccountID string
	subAccountID    string
	runtimeID       string
	outputPath      string
}

// NewKubeconfigCmd constructs a new instance of KubeconfigCommand and configures it in terms of a cobra.Command
func NewKubeconfigCmd(logger logger.Logger) *cobra.Command {
	cmd := KubeconfigCommand{logger: logger}
	cobraCmd := &cobra.Command{
		Use:     "kubeconfig",
		Aliases: []string{"kc"},
		Short:   "Download kubeconfig for given Kyma runtime",
		Long: `Downloads kubeconfig for given Kyma runtime.
The runtime can be specified by either of the following:
  - Global Account / Subaccount pair with the --account and --subaccount options
  - Global Account / Runtime ID pair with the --account and --runtime-id options
  - Shoot cluster name with the --shoot option

By default the kubeconfig is saved to the current directory. The output file name can be specified using the --output option.`,
		Example: `  skr kubeconfig -g GAID -s SAID -o /my/path/skr.config  Download kubeconfig using Global Account ID and Subaccount ID
  skr kubeconfig -g GAID -r RUNTIMEID                    Download kubeconfig using Global Account ID and Runtime ID
  skr kubeconfig -c c-178e034                            Download kubeconfig using Shoot cluster name`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}

	cobraCmd.Flags().StringVarP(&cmd.outputPath, "output", "o", "", "Path to the file to save the downloaded kubeconfig to. Defaults to <CLUSTER NAME>.yaml in the current directory if not specified.")
	cobraCmd.Flags().StringVarP(&cmd.globalAccountID, "account", "g", "", "Global Account ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.subAccountID, "subaccount", "s", "", "Subccount ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.shoot, "shoot", "c", "", "Shoot cluster name of the specific Kyma Runtime")

	return cobraCmd
}

// Run executes the kubeconfig command
func (cmd *KubeconfigCommand) Run() error {
	fmt.Println("Not implemented yet.")
	return nil
}

// Validate checks the input parameters of the kubeconfig command
func (cmd *KubeconfigCommand) Validate() error {
	// TODO: implement
	return nil
}
