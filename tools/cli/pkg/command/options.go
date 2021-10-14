package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string

const (
	configEnv string = "KCPCONFIG"
	configDir string = ".kcp"
)

const (
	tableOutput  string = "table"
	jsonOutput   string = "json"
	customOutput string = "custom"
)

const (
	accountTarget    = "account"
	subaccountTarget = "subaccount"
	runtimeIDTarget  = "runtime-id"
	instanceIDTarget = "instance-id"
	regionTarget     = "region"
	planTarget       = "plan"
	shootTarget      = "shoot"
)

const (
	azurePlan     = "azure"
	azureLitePlan = "azure_lite"
	azureHAPlan   = "azure_ha"
	trialPlan     = "trial"
	gcpPlan       = "gcp"
	openstackPlan = "openstack"
	awsPlan       = "aws"
	awsHAPlan     = "aws_ha"
	freePlan      = "free"
)

// GlobalOptionsKey is the type for holding the configuration key for each global parameter
type GlobalOptionsKey struct {
	oidcIssuerURL      string
	oidcClientID       string
	oidcClientSecret   string
	kebAPIURL          string
	mothershipAPIURL   string
	kubeconfigAPIURL   string
	gardenerKubeconfig string
	gardenerNamespace  string
	username           string
}

// GlobalOpts is the convenience object for storing the fixed global conifguration (parameter) keys
var GlobalOpts = GlobalOptionsKey{
	oidcIssuerURL:      "oidc-issuer-url",
	oidcClientID:       "oidc-client-id",
	oidcClientSecret:   "oidc-client-secret",
	kebAPIURL:          "keb-api-url",
	mothershipAPIURL:   "mothership-api-url",
	kubeconfigAPIURL:   "kubeconfig-api-url",
	gardenerKubeconfig: "gardener-kubeconfig",
	gardenerNamespace:  "gardener-namespace",
	username:           "username",
}

// SetGlobalOpts configures the global parameters on the given root command
func SetGlobalOpts(cmd *cobra.Command) {
	cmd.PersistentFlags().String(GlobalOpts.oidcIssuerURL, "", "OIDC authentication server URL to use for login. Can also be set using the KCP_OIDC_ISSUER_URL environment variable.")
	viper.BindPFlag(GlobalOpts.oidcIssuerURL, cmd.PersistentFlags().Lookup(GlobalOpts.oidcIssuerURL))

	cmd.PersistentFlags().String(GlobalOpts.oidcClientID, "", "OIDC client ID to use for login. Can also be set using the KCP_OIDC_CLIENT_ID environment variable.")
	viper.BindPFlag(GlobalOpts.oidcClientID, cmd.PersistentFlags().Lookup(GlobalOpts.oidcClientID))

	cmd.PersistentFlags().String(GlobalOpts.oidcClientSecret, "", "OIDC client secret to use for login. Can also be set using the KCP_OIDC_CLIENT_SECRET environment variable.")
	viper.BindPFlag(GlobalOpts.oidcClientSecret, cmd.PersistentFlags().Lookup(GlobalOpts.oidcClientSecret))

	cmd.PersistentFlags().String(GlobalOpts.kebAPIURL, "", "Kyma Environment Broker API URL to use for all commands. Can also be set using the KCP_KEB_API_URL environment variable.")
	viper.BindPFlag(GlobalOpts.kebAPIURL, cmd.PersistentFlags().Lookup(GlobalOpts.kebAPIURL))

	cmd.PersistentFlags().String(GlobalOpts.mothershipAPIURL, "", "Mothership API URL to use for all commands. Can also be set using the KCP_MOTHERSHIP_API_URL environment variable.")
	viper.BindPFlag(GlobalOpts.mothershipAPIURL, cmd.PersistentFlags().Lookup(GlobalOpts.mothershipAPIURL))

	cmd.PersistentFlags().String(GlobalOpts.kubeconfigAPIURL, "", "OIDC Kubeconfig Service API URL used by the kcp kubeconfig and taskrun commands. Can also be set using the KCP_KUBECONFIG_API_URL environment variable.")
	viper.BindPFlag(GlobalOpts.kubeconfigAPIURL, cmd.PersistentFlags().Lookup(GlobalOpts.kubeconfigAPIURL))

	cmd.PersistentFlags().String(GlobalOpts.gardenerKubeconfig, "", "Path to the kubeconfig file of the corresponding Gardener project which has permissions to list/get Shoots. Can also be set using the KCP_GARDENER_KUBECONFIG environment variable.")
	viper.BindPFlag(GlobalOpts.gardenerKubeconfig, cmd.PersistentFlags().Lookup(GlobalOpts.gardenerKubeconfig))

	cmd.PersistentFlags().String(GlobalOpts.gardenerNamespace, "", "Gardener Namespace (project) to use. Can also be set using the KCP_GARDENER_NAMESPACE environment variable.")
	viper.BindPFlag(GlobalOpts.gardenerNamespace, cmd.PersistentFlags().Lookup(GlobalOpts.gardenerNamespace))

	viper.BindEnv(GlobalOpts.username)
}

// ValidateGlobalOpts checks the presence of the required global configuration parameters
func ValidateGlobalOpts() error {
	var reqGlobalOpts = []string{GlobalOpts.oidcIssuerURL, GlobalOpts.oidcClientID, GlobalOpts.kebAPIURL}
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

// MothershipAPIURL gets the mothership-api-url global parameter
func (keys *GlobalOptionsKey) MothershipAPIURL() string {
	return viper.GetString(keys.mothershipAPIURL)
}

// KubeconfigAPIURL gets the kubeconfig-api-url global parameter
func (keys *GlobalOptionsKey) KubeconfigAPIURL() string {
	return viper.GetString(keys.kubeconfigAPIURL)
}

// GardenerKubeconfig gets the gardener-kubeconfig global parameter
func (keys *GlobalOptionsKey) GardenerKubeconfig() string {
	return viper.GetString(keys.gardenerKubeconfig)
}

// GardenerNamespace gets the gardener-namespace global parameter
func (keys *GlobalOptionsKey) GardenerNamespace() string {
	return viper.GetString(keys.gardenerNamespace)
}

// Username gets the username to use for auth
func (keys *GlobalOptionsKey) Username() string {
	return viper.GetString(keys.username)
}

// SetOutputOpt configures the optput type option on the given command
func SetOutputOpt(cmd *cobra.Command, opt *string) {
	cmd.Flags().StringVarP(opt, "output", "o", tableOutput, fmt.Sprintf("Output type of displayed Runtime(s). The possible values are: %s, %s, %s(e.g. custom=<header>:<jsonpath-field-spec>.", tableOutput, jsonOutput, customOutput))
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

// SetRuntimeTargetOpts configures runtime target options on the given command
func SetRuntimeTargetOpts(cmd *cobra.Command, targetInputs *[]string, targetExcludeInputs *[]string) {
	cmd.Flags().StringArrayVarP(targetInputs, "target", "t", nil,
		`List of Runtime target specifiers to include. You can specify this option multiple times.
A target specifier is a comma-separated list of the following selectors:
  all                 : All Runtimes provisioned successfully and not deprovisioning
  account={REGEXP}    : Regex pattern to match against the Runtime's global account field, e.g. "CA50125541TID000000000741207136", "CA.*"
  subaccount={REGEXP} : Regex pattern to match against the Runtime's subaccount field, e.g. "0d20e315-d0b4-48a2-9512-49bc8eb03cd1"
  region={REGEXP}     : Regex pattern to match against the Runtime's provider region field, e.g. "europe|eu-"
  runtime-id={ID}     : Specific Runtime by Runtime ID
  plan={NAME}         : Name of the Runtime's service plan. The possible values are: azure, azure_lite, aws, trial, gcp, openstack
  shoot={NAME}        : Specific Runtime by Shoot cluster name
  instance-id={ID}    : Specific instance by Instance ID`)
	cmd.Flags().StringArrayVarP(targetExcludeInputs, "target-exclude", "e", nil,
		`List of Runtime target specifiers to exclude. You can specify this option multiple times.
A target specifier is a comma-separated list of the selectors described under the --target option.`)
}

// ValidateTransformRuntimeTargetOpts checks the validity of runtime target options, and transforms them for internal usage
func ValidateTransformRuntimeTargetOpts(targetInputs []string, targetExcludeInputs []string, targetSpec *orchestration.TargetSpec) error {
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

func parseRuntimeTarget(targetInput string, targets *[]orchestration.RuntimeTarget, include bool) error {
	target := orchestration.RuntimeTarget{}
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

		err := checkMissingRuntimeTargetSelector(selectorKey, selectorValue, flagName)
		if err != nil {
			return err
		}
		switch selectorKey {
		case orchestration.TargetAll:
			if !include {
				return fmt.Errorf("\"%s\" cannot be used with --target-exclude", orchestration.TargetAll)
			}
			target.Target = orchestration.TargetAll
		case accountTarget:
			target.GlobalAccount = selectorValue
		case subaccountTarget:
			target.SubAccount = selectorValue
		case regionTarget:
			target.Region = selectorValue
		case runtimeIDTarget:
			target.RuntimeID = selectorValue
		case instanceIDTarget:
			target.InstanceID = selectorValue
		case planTarget:
			switch selectorValue {
			case azurePlan, azureLitePlan, azureHAPlan, trialPlan, gcpPlan, openstackPlan, awsPlan, awsHAPlan, freePlan:
				target.PlanName = selectorValue
			default:
				return fmt.Errorf("invalid value for selector: %s %s=%s", flagName, selectorKey, selectorValue)
			}
		case shootTarget:
			target.Shoot = selectorValue
		default:
			return fmt.Errorf("invalid selector: %s %s", flagName, selectorKey)
		}
	}

	*targets = append(*targets, target)
	return nil
}

func checkMissingRuntimeTargetSelector(selectorKey, selectorValue string, flagName string) error {

	if selectorKey != orchestration.TargetAll && selectorValue == "" {
		return fmt.Errorf("%s %s is missing required value (%s={VALUE})", flagName, selectorKey, selectorKey)
	}

	return nil
}
