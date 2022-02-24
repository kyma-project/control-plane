package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string

const (
	configEnv string = "ERSCONFIG"
	configDir string = ".ers"
)

const (
	tableOutput  string = "table"
	jsonOutput   string = "json"
	customOutput string = "custom"
)

// GlobalOptionsKey is the type for holding the configuration key for each global parameter
type GlobalOptionsKey struct {
	clientId     string
	clientSecret string
	oauthUrl     string
	ersUrl       string
}

// GlobalOpts is the convenience object for storing the fixed global conifguration (parameter) keys
var GlobalOpts = GlobalOptionsKey{
	clientId:     "client-id",
	clientSecret: "client-secret",
	oauthUrl:     "oauth-url",
	ersUrl:       "ers-url",
}

func (key *GlobalOptionsKey) ClientID() string {
	return viper.GetString(key.clientId)
}

func (key *GlobalOptionsKey) ClientSecret() string {
	return viper.GetString(key.clientSecret)
}

func (key *GlobalOptionsKey) ErsUrl() string {
	return viper.GetString(key.ersUrl)
}

func (key *GlobalOptionsKey) OauthUrl() string {
	return viper.GetString(key.oauthUrl)
}

// SetGlobalOpts configures the global parameters on the given root command
func SetGlobalOpts(cmd *cobra.Command) {
	//fmt.Println(configPath)
	//cmd.PersistentFlags().String(GlobalOpts.clientId, "", "Client ID")
	//viper.BindPFlag(GlobalOpts.clientId, cmd.PersistentFlags().Lookup(GlobalOpts.clientId))
	//
	//cmd.PersistentFlags().String(GlobalOpts.clientSecret, "", "Client Secret")
	//viper.BindPFlag(GlobalOpts.clientSecret, cmd.PersistentFlags().Lookup(GlobalOpts.clientSecret))
	//
	//cmd.PersistentFlags().String(GlobalOpts.ersUrl, "", "ERS API URL")
	//viper.BindPFlag(GlobalOpts.ersUrl, cmd.PersistentFlags().Lookup(GlobalOpts.ersUrl))
	//
	//cmd.PersistentFlags().String(GlobalOpts.oauthUrl, "", "ERS Oauth URL to use for all commands.")
	//viper.BindPFlag(GlobalOpts.oauthUrl, cmd.PersistentFlags().Lookup(GlobalOpts.oauthUrl))
}

// ValidateGlobalOpts checks the presence of the required global configuration parameters
func ValidateGlobalOpts() error {
	var reqGlobalOpts = []string{GlobalOpts.clientSecret, GlobalOpts.clientId, GlobalOpts.ersUrl, GlobalOpts.oauthUrl}
	var missingGlobalOpts []string
	for _, opt := range reqGlobalOpts {
		if viper.GetString(opt) == "" {
			missingGlobalOpts = append(missingGlobalOpts, opt)
		}
	}

	if len(missingGlobalOpts) == 0 {
		return nil
	}
	return fmt.Errorf("missing required options: %s. See kcp --help for more information", strings.Join(missingGlobalOpts, ", "))
}

// SetOutputOpt configures the optput type option on the given command
func SetOutputOpt(cmd *cobra.Command, opt *string) {
	cmd.Flags().StringVarP(opt, "output", "o", tableOutput, fmt.Sprintf("Output type of displayed Instances(s). The possible values are: %s, %s, %s(e.g. custom=<header>:<jsonpath-field-spec>.", tableOutput, jsonOutput, customOutput))
}

// ValidateOutputOpt checks whether the given optput type is one of the valid values
func ValidateOutputOpt(opt string) error {
	switch {
	case opt == tableOutput, opt == jsonOutput:
		return nil
	case strings.HasPrefix(opt, customOutput):
		return nil
	}
	return fmt.Errorf("invalid value for output: %s", opt)
}
