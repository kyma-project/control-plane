package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/credential"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the CLI version to be filled in by the build system
var Version string = "N/A"

// New constructs a new root command for the kcp CLI.
func New(log logger.Logger) *cobra.Command {
	cobra.OnInitialize(initConfig)
	description := fmt.Sprintf(`The kcp CLI (a.k.a. Kyma Control Plane CLI) is a day-two operations tool for Kyma runtimes, which allows to view and manage the runtimes in scale.
It is possible to list and observe attributes and state of each Kyma runtime and perform various operations on them, e.g. upgrading the Kyma version.
You can find the complete list of possible operations as commands below.

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:
  --config <PATH> option
  KCPCONFIG environment variable which contains the path
  $HOME/.kcp/config.yaml (default path)

The configuration file is in YAML format and supports the following global options: %s, %s, %s, %s, %s, %s.`, GlobalOpts.oidcIssuerURL, GlobalOpts.oidcClientID, GlobalOpts.oidcClientSecret, GlobalOpts.kebAPIURL, GlobalOpts.kubeconfigAPIURL, GlobalOpts.gardenerKubeconfig)

	cmd := &cobra.Command{
		Use:     "kcp",
		Short:   "Day-two operations tool for Kyma Runtimes",
		Long:    description,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.CalledAs() != "help" {
				return ValidateGlobalOpts()
			}
			return nil
		},
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", os.Getenv(configEnv), "Path to the kcp CLI config file. Can also be set via the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml")
	SetGlobalOpts(cmd)
	log.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().BoolP("help", "h", false, "Displays help for the CLI")

	cmd.AddCommand(
		NewLoginCmd(log),
		NewRuntimeCmd(log),
		NewOrchestrationCmd(log),
		NewKubeconfigCmd(log),
		NewUpgradeCmd(log),
		NewTaskRunCmd(log),
	)
	return cmd
}

func initConfig() {
	// If config file is set via flags or ENV, use that path,
	// otherwise try to load the config from $HOME/{configDir}/config.yaml
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		configPath = fmt.Sprintf("%s/%s", home, configDir)
		viper.AddConfigPath(configPath)
		viper.SetConfigName("config")
	}
	viper.SetConfigType("yaml")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("KCP")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	// Ignore when config file is not found to allow config parameters being passed as flags or environment variables
	// Panic otherwise
	if _, ok := err.(viper.ConfigFileNotFoundError); err != nil && !ok {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// CLICredentialManager returns a credential.Manager configured using the CLI global options
func CLICredentialManager(logger logger.Logger) credential.Manager {
	return credential.NewManager(GlobalOpts.OIDCIssuerURL(), GlobalOpts.OIDCClientID(), GlobalOpts.OIDCClientSecret(), logger)
}
