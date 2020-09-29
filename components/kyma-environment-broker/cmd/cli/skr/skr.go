package skr

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the CLI version to be filled in by the build system
var Version string = "N/A"

// NewCmd constructs a new root command for the skr CLI.
func NewCmd(logger logger.Logger) *cobra.Command {
	cobra.OnInitialize(initConfig)
	description := fmt.Sprintf(`The skr CLI is a day-two operations tool for SAP CP Kyma Runtimes (SKRs), which allows to view and manage SKRs in scale.
It is possible to list and observe attributes and state of each SKRs and perform various operations on them, e.g. upgrading the Kyma version.
You can find the complete list of possible operations as commands below.

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:
  --config <PATH> option
  SKRCONFIG environment variable which contains the path
  $HOME/.skr/config.yaml (default path)

The configuration file is in YAML format and supports the following global options: %s, %s, %s, %s, %s, %s.`, GlobalOpts.oidcIssuerURL, GlobalOpts.oidcClientID, GlobalOpts.oidcClientSecret, GlobalOpts.kebAPIURL, GlobalOpts.kubeconfigAPIURL, GlobalOpts.gardenerKubeconfig)

	cmd := &cobra.Command{
		Use:     "skr",
		Short:   "Day-two operations tool for SAP CP Kyma Runtimes (SKRs)",
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

	cmd.PersistentFlags().StringVar(&configPath, "config", os.Getenv(configEnv), "Path to the skr CLI config file. Can also be set via the SKRCONFIG environment variable. Defaults to $HOME/.skr/config.yaml")
	SetGlobalOpts(cmd)
	logger.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().BoolP("help", "h", false, "Displays help for the CLI")

	cmd.AddCommand(
		NewLoginCmd(logger),
		NewRuntimeCmd(logger),
		NewOrchestrationCmd(logger),
		NewKubeconfigCmd(logger),
		NewUpgradeCmd(logger),
		NewTaskRunCmd(logger),
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
	viper.SetEnvPrefix("SKR")
	viper.AutomaticEnv()
	viper.ReadInConfig()
}
