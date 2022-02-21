package command

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2/clientcredentials"
)

func New() *cobra.Command {
	cobra.OnInitialize(initConfig)
	cmd := &cobra.Command{
		Use:              "ers",
		Short:            "ERS operations tool for Kyma Runtimes.",
		Long:             "The ers tool provides commands to play with ERS API for Service Catalog migration.",
		Version:          "N/A",
		SilenceUsage:     true,
		TraverseChildren: true,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", os.Getenv(configEnv), "Path to the ERS CLI config file. Can also be set using the ERSCONFIG environment variable.")
	SetGlobalOpts(cmd)
	logger.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().BoolP("help", "h", false, "Option that displays help for the CLI.")

	cmd.AddCommand(NewInstancesCommand(), NewMigrationCommand(), NewSwitchBrokerCommand())

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
	viper.SetEnvPrefix("ERS")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()

	// Ignore when config file is not found to allow config parameters being passed as flags or environment variables
	// Panic otherwise
	if _, ok := err.(viper.ConfigFileNotFoundError); err != nil && !ok {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	config := clientcredentials.Config{
		ClientID:     GlobalOpts.ClientID(),
		ClientSecret: GlobalOpts.ClientSecret(),
		TokenURL:     GlobalOpts.OauthUrl(),
	}

	// create a shared ERS HTTP client which does the oauth flow
	ErsHttpClient = config.Client(context.Background())
}
