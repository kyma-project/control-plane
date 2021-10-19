package command

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/client"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/credential"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// KubeconfigCommand represents an execution of the kcp kubeconfig command
type KubeconfigCommand struct {
	cobraCmd        *cobra.Command
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
func NewKubeconfigCmd() *cobra.Command {
	cmd := KubeconfigCommand{}
	cobraCmd := &cobra.Command{
		Use:     "kubeconfig",
		Aliases: []string{"kc"},
		Short:   "Downloads the kubeconfig file for a given Kyma Runtime",
		Long: `Downloads the kubeconfig file for a given Kyma Runtime.
The Runtime can be specified by one of the following:
  - Global account / subaccount pair with the --account and --subaccount options
  - Global account / Runtime ID pair with the --account and --runtime-id options
  - Shoot cluster name with the --shoot option.

By default, the kubeconfig file is saved to the current directory. The output file name can be specified using the --output option.`,
		Example: `  kcp kubeconfig -g GAID -s SAID -o /my/path/runtime.config  Downloads the kubeconfig file using global account ID and subaccount ID.
  kcp kubeconfig -g GAID -r RUNTIMEID                    Downloads the kubeconfig file using global account ID and Runtime ID.
  kcp kubeconfig -c c-178e034                            Downloads the kubeconfig file using a Shoot cluster name.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.outputPath, "output", "o", "", "Path to the file to save the downloaded kubeconfig to. Defaults to {CLUSTER NAME}.yaml in the current directory if not specified.")
	cobraCmd.Flags().StringVarP(&cmd.globalAccountID, "account", "g", "", "Global account ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.subAccountID, "subaccount", "s", "", "Subccount ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.runtimeID, "runtime-id", "r", "", "Runtime ID of the specific Kyma Runtime.")
	cobraCmd.Flags().StringVarP(&cmd.shoot, "shoot", "c", "", "Shoot cluster name of the specific Kyma Runtime.")

	return cobraCmd
}

// Run executes the kubeconfig command
func (cmd *KubeconfigCommand) Run() error {
	cmd.log = logger.New()
	cred := CLICredentialManager(cmd.log)
	client := client.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KubeconfigAPIURL(), cred)

	// Resolve Global Account / Subaccount, or Shoot name to Global Account / Runtime ID
	if cmd.globalAccountID == "" || cmd.runtimeID == "" {
		err := cmd.resolveRuntimeAttributes(cmd.cobraCmd.Context(), cred)
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
	httpClient := oauth2.NewClient(ctx, cred)
	rtClient := runtime.NewClient(GlobalOpts.KEBAPIURL(), httpClient)
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
