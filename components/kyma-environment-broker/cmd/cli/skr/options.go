package skr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string

const (
	configEnv string = "SKRCONFIG"
	configDir string = ".skr"
)

const (
	table string = "table"
	json  string = "json"
	yaml  string = "yaml"
)

// GlobalOptionsKey is the type for holding the configuration key for each global parameter
type GlobalOptionsKey struct {
	oidcIssuerURL      string
	oidcClientID       string
	oidcClientSecret   string
	kebAPIURL          string
	kubeconfigAPIURL   string
	gardenerKubeconfig string
}

// GlobalOpts is the convenience object for storing the fixed global conifguration (parameter) keys
var GlobalOpts = GlobalOptionsKey{
	oidcIssuerURL:      "oidc-issuer-url",
	oidcClientID:       "oidc-client-id",
	oidcClientSecret:   "oidc-client-secret",
	kebAPIURL:          "keb-api-url",
	kubeconfigAPIURL:   "kubeconfig-api-url",
	gardenerKubeconfig: "gardener-kubeconfig",
}

// SetGlobalOpts configures the global parameters on the given root command
func SetGlobalOpts(cmd *cobra.Command) {
	cmd.PersistentFlags().String(GlobalOpts.oidcIssuerURL, "", "OIDC authentication server URL to use for login. Can also be set the SKR_OIDC_ISSUER_URL environment variable")
	viper.BindPFlag(GlobalOpts.oidcIssuerURL, cmd.PersistentFlags().Lookup(GlobalOpts.oidcIssuerURL))

	cmd.PersistentFlags().String(GlobalOpts.oidcClientID, "", "OIDC client ID to use for login. Can also be set via the SKR_OIDC_CLIENT_ID environment variable")
	viper.BindPFlag(GlobalOpts.oidcClientID, cmd.PersistentFlags().Lookup(GlobalOpts.oidcClientID))

	cmd.PersistentFlags().String(GlobalOpts.oidcClientSecret, "", "OIDC client Secret to use for login. Can also be set via the SKR_OIDC_CLIENT_SECRET environment variable")
	viper.BindPFlag(GlobalOpts.oidcClientSecret, cmd.PersistentFlags().Lookup(GlobalOpts.oidcClientSecret))

	cmd.PersistentFlags().String(GlobalOpts.kebAPIURL, "", "Kyma Environment Broker API URL to use for all commands. Can also be set via the SKR_KEB_API_URL environment variable")
	viper.BindPFlag(GlobalOpts.kebAPIURL, cmd.PersistentFlags().Lookup(GlobalOpts.kebAPIURL))

	cmd.PersistentFlags().String(GlobalOpts.kubeconfigAPIURL, "", "OIDC Kubeconfig Service API URL, used by the skr kubeconfig and taskrun commands. Can also be set via the SKR_KUBECONFIG_API_URL environment variable")
	viper.BindPFlag(GlobalOpts.kubeconfigAPIURL, cmd.PersistentFlags().Lookup(GlobalOpts.kubeconfigAPIURL))

	cmd.PersistentFlags().String(GlobalOpts.gardenerKubeconfig, "", "Path to the corresponding Gardener project kubeconfig file which have permissions to list/get shoots. Can also be set via the SKR_GARDENER_KUBECONFIG environment variable")
	viper.BindPFlag(GlobalOpts.gardenerKubeconfig, cmd.PersistentFlags().Lookup(GlobalOpts.gardenerKubeconfig))
}

// ValidateGlobalOpts checks the presence of the required global configuration parameters
func ValidateGlobalOpts() error {
	var reqGlobalOpts = []string{GlobalOpts.oidcIssuerURL, GlobalOpts.oidcClientID, GlobalOpts.oidcClientSecret, GlobalOpts.kebAPIURL}
	var missingGlobalOpts []string
	for _, opt := range reqGlobalOpts {
		if viper.GetString(opt) == "" {
			missingGlobalOpts = append(missingGlobalOpts, opt)
		}
	}

	if len(missingGlobalOpts) == 0 {
		return nil
	}
	return fmt.Errorf("missing required options: %s. See skr --help for more information", strings.Join(missingGlobalOpts, ", "))
}

// OIDCIssuerURL gets the oidc-issuer-url global parameter
func (keys *GlobalOptionsKey) OIDCIssuerURL() string {
	return viper.GetString(keys.oidcIssuerURL)
}

// OIDCClientID gets the oidc-client-id global parameter
func (keys *GlobalOptionsKey) OIDCClientID() string {
	return viper.GetString(keys.oidcClientID)
}

// OIDCClientSecret gets the oidc-client-secret global parameter
func (keys *GlobalOptionsKey) OIDCClientSecret() string {
	return viper.GetString(keys.oidcClientSecret)
}

// KEBAPIURL gets the keb-api-url global parameter
func (keys *GlobalOptionsKey) KEBAPIURL() string {
	return viper.GetString(keys.kebAPIURL)
}

// KubeconfigAPIURL gets the kubeconfig-api-url global parameter
func (keys *GlobalOptionsKey) KubeconfigAPIURL() string {
	return viper.GetString(keys.kubeconfigAPIURL)
}

// GardenerKubeconfig gets the gardener-kubeconfig global parameter
func (keys *GlobalOptionsKey) GardenerKubeconfig() string {
	return viper.GetString(keys.gardenerKubeconfig)
}

// SetOutputOpt configures the optput type option on the given command
func SetOutputOpt(cmd *cobra.Command, opt *string) {
	cmd.Flags().StringVarP(opt, "output", "o", table, fmt.Sprintf("Output type of displayed runtime(s). Possible values: %s, %s, %s", table, json, yaml))
}

// ValidateOutputOpt checks whether the given optput type is one of the valid values
func ValidateOutputOpt(opt string) error {
	switch opt {
	case table, json, yaml:
		return nil
	}
	return fmt.Errorf("invalid value for output: %s", opt)
}

// SetRuntimeTargetOpts configures runtime target options on the given command
func SetRuntimeTargetOpts(cmd *cobra.Command, targetInputs *[]string, targetExcludeInputs *[]string) {
	cmd.Flags().StringArrayVarP(targetInputs, "target", "t", nil,
		`List of runtime target specifiers to include (the option can be specified multiple times).
A target specifier is a comma separated list of the following selectors:
  all                 : all SKRs provisioned successfully and not deprovisioning
  account=<REGEXP>    : Regex pattern to match against the runtime's GlobalAccount field. E.g. CA50125541TID000000000741207136, "CA.*"
  subaccount=<REGEXP> : Regex pattern to match against the runtime's SubAccount field. E.g. 0d20e315-d0b4-48a2-9512-49bc8eb03cd1
  region=<REGEXP>     : Regex pattern to match against the shoot cluster's Region field (not SCP platform-region). E.g. "europe|eu-"
  runtime-id=<ID>     : Runtime ID is used to indicate a specific runtime`)
	cmd.Flags().StringArrayVarP(targetExcludeInputs, "target-exclude", "e", nil,
		`List of runtime target specifiers to exclude (the option can be specified multiple times).
A target specifier is a comma separated list of the selectors described under --target option`)
}

// ValidateTransformRuntimeTargetOpts checks the validity of runtime target options, and transforms them for internal usage
func ValidateTransformRuntimeTargetOpts(targetInputs []string, targetExcludeInputs []string, targetSpec *internal.TargetSpec) error {
	if len(targetInputs) == 0 {
		return errors.New("at least one runtime target must be specified with --target")
	}
	for _, target := range targetInputs {
		err := parseRuntimeTarget(target, &targetSpec.Include, true)
		if err != nil {
			return err
		}
	}
	for _, target := range targetExcludeInputs {
		err := parseRuntimeTarget(target, &targetSpec.Exclude, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseRuntimeTarget(targetInput string, targets *[]internal.RuntimeTarget, include bool) error {
	target := internal.RuntimeTarget{}
	selectors := strings.Split(targetInput, ",")
	var flagName string
	if include {
		flagName = "--target"
	} else {
		flagName = "--target-exclude"
	}

	for _, selector := range selectors {
		sv := strings.Split(selector, "=")
		selectorKey := sv[0]
		var selectorValue string
		if len(sv) > 1 {
			selectorValue = sv[1]
		} else {
			selectorValue = ""
		}
		switch selectorKey {
		case internal.TargetAll:
			if !include {
				return errors.New("\"all\" cannot be used with --target-exclude")
			}
			target.Target = internal.TargetAll
		case "account":
			err := checkRuntimeTargetSelector(selectorKey, selectorValue, flagName)
			if err != nil {
				return err
			}
			target.GlobalAccount = selectorValue
		case "subaccount":
			err := checkRuntimeTargetSelector(selectorKey, selectorValue, flagName)
			if err != nil {
				return err
			}
			target.SubAccount = selectorValue
		case "region":
			err := checkRuntimeTargetSelector(selectorKey, selectorValue, flagName)
			if err != nil {
				return err
			}
			target.Region = selectorValue
		case "runtime-id":
			err := checkRuntimeTargetSelector(selectorKey, selectorValue, flagName)
			if err != nil {
				return err
			}
			target.RuntimeID = selectorValue
		default:
			return fmt.Errorf("invalid selector: %s %s", flagName, selectorKey)
		}
	}

	*targets = append(*targets, target)
	return nil
}

func checkRuntimeTargetSelector(selectorKey, selectorValue string, flagName string) error {

	if selectorValue == "" {
		return fmt.Errorf("%s %s is missing required value (--target %s=<VALUE>)", flagName, selectorKey, selectorKey)
	}

	return nil
}
