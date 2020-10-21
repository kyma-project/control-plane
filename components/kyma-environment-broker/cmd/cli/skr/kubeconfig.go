package skr

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/client"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/credential"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// KubeconfigCommand represents an execution of the skr kubeconfig command
type KubeconfigCommand struct {
	log             logger.Logger
	shoot           string
	globalAccountID string
	subAccountID    string
	runtimeID       string
	outputPath      string
}

type kubeconfig struct {
	APIVersion     string `yaml:"apiVersion"`
	Kind           string `yaml:"kind"`
	CurrentContext string `yaml:"current-context"`
	Clusters       []struct {
		Name string `yaml:"name"`
	} `yaml:"clusters"`
}

// NewKubeconfigCmd constructs a new instance of KubeconfigCommand and configures it in terms of a cobra.Command
func NewKubeconfigCmd(log logger.Logger) *cobra.Command {
	cmd := KubeconfigCommand{log: log}
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
		RunE:    func(cobraCmd *cobra.Command, _ []string) error { return cmd.Run(cobraCmd) },
	}

	cobraCmd.Flags().StringVarP(&cmd.outputPath, "output", "o", "", "Path to the file to save the downloaded kubeconfig to. Defaults to <CLUSTER NAME>.yaml in the current directory if not specified.")
	cobraCmd.Flags().StringVarP(&cmd.globalAccountID, "account", "g", "", "Global Account ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.subAccountID, "subaccount", "s", "", "Subccount ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime")
	cobraCmd.Flags().StringVarP(&cmd.shoot, "shoot", "c", "", "Shoot cluster name of the specific Kyma Runtime")

	return cobraCmd
}

// Run executes the kubeconfig command
func (cmd *KubeconfigCommand) Run(cobraCmd *cobra.Command) error {
	cred := CLICredentialManager(cmd.log)
	client := client.NewClient(cobraCmd.Context(), GlobalOpts.KubeconfigAPIURL(), cred)

	// Resolve Global Account / Subaccount, or Shoot name to Global Account / Runtime ID
	if cmd.globalAccountID == "" || cmd.runtimeID == "" {
		err := cmd.resolveRuntimeAttributes(cobraCmd.Context(), cred)
		if err != nil {
			return errors.Wrap(err, "while resolving runtime")
		}
	}
	kc, err := client.GetKubeConfig(cmd.globalAccountID, cmd.runtimeID)
	if err != nil {
		return errors.Wrap(err, "while getting kubeconfig")
	}
	err = cmd.saveKubeconfig(kc)
	return err
}

// Validate checks the input parameters of the kubeconfig command
func (cmd *KubeconfigCommand) Validate() error {
	if GlobalOpts.KubeconfigAPIURL() == "" {
		return fmt.Errorf("missing required %s option", GlobalOpts.kubeconfigAPIURL)
	}
	if cmd.globalAccountID != "" && (cmd.subAccountID != "" || cmd.runtimeID != "") || cmd.shoot != "" {
		return nil
	}
	return errors.New("at least one of the following options have to be specified: account/subaccount, account/runtime-id, shoot")
}

func (cmd *KubeconfigCommand) resolveRuntimeAttributes(ctx context.Context, cred credential.Manager) error {
	rtClient := runtime.NewClient(ctx, GlobalOpts.KEBAPIURL(), cred)
	params := runtime.ListParameters{}
	if cmd.shoot != "" {
		params.Shoots = []string{cmd.shoot}
	} else {
		params.GlobalAccountIDs = []string{cmd.globalAccountID}
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
	cmd.globalAccountID = rp.Data[0].GlobalAccountID
	return nil
}

func (cmd *KubeconfigCommand) saveKubeconfig(kubeconfig string) error {
	// Assemble default output path based on cluster name if output path was not given
	if cmd.outputPath == "" {
		clusterName, err := clusterNameFromKubeconfig(kubeconfig)
		if err != nil {
			return errors.Wrap(err, "while getting cluster name from kubeconfig")
		}
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "while getting current directory")
		}
		cmd.outputPath = fmt.Sprintf("%s/%s.yaml", dir, clusterName)
	}

	err := ioutil.WriteFile(cmd.outputPath, []byte(kubeconfig), 0600)
	if err != nil {
		return errors.Wrap(err, "while saving kubeconfig")
	}
	fmt.Printf("Kubeconfig saved to %s\n", cmd.outputPath)

	return nil
}

func clusterNameFromKubeconfig(rawKubeConfig string) (string, error) {
	var kubeCfg kubeconfig
	var clusterName string
	err := yaml.Unmarshal([]byte(rawKubeConfig), &kubeCfg)
	if err != nil {
		return "", err
	}
	if len(kubeCfg.Clusters) > 0 {
		clusterName = kubeCfg.Clusters[0].Name
	} else {
		clusterName = kubeCfg.CurrentContext
	}

	return clusterName, nil
}
