package runtimeoverrides

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace                      = "kcp-system"
	componentNameLabel             = "component"
	overridesVersionLabelPrefix    = "overrides-version-"
	overridesPlanLabelPrefix       = "overrides-plan-"
	overridesAccountLabelPrefix    = "overrides-account-"
	overridesSubaccountLabelPrefix = "overrides-subaccount-"
	overridesSecretLabel           = "runtime-override"
	PLANNAME                       = "planeName"
	ACCOUNT                        = "account"
	SUBACCOUNT                     = "subaccount"
)

type InputAppender interface {
	AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator
	AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator
}

type runtimeOverrides struct {
	ctx       context.Context
	log       logrus.FieldLogger
	k8sClient client.Client
}

func NewRuntimeOverrides(ctx context.Context, log logrus.FieldLogger, cli client.Client) *runtimeOverrides {
	return &runtimeOverrides{
		ctx:       ctx,
		log:       log,
		k8sClient: cli,
	}
}

func (ro *runtimeOverrides) Append(input InputAppender, planName, overridesVersion, account, subaccount string) error {
	{
		componentsOverrides, globalOverrides, err := ro.collectFromSecrets()
		if err != nil {
			return err
		}

		appendOverrides(input, componentsOverrides, globalOverrides)
	}

	{
		componentsOverrides, globalOverrides, err := ro.collectFromConfigMaps(planName, overridesVersion, account, subaccount)
		if err != nil {
			return err
		}

		if len(globalOverrides) == 0 {
			return fmt.Errorf("no global overrides for plan '%s' and version '%s'", planName, overridesVersion)
		}

		appendOverrides(input, componentsOverrides, globalOverrides)
	}

	return nil
}

func (ro *runtimeOverrides) collectFromSecrets() (map[string][]*gqlschema.ConfigEntryInput, []*gqlschema.ConfigEntryInput, error) {
	componentsOverrides := make(map[string][]*gqlschema.ConfigEntryInput, 0)
	globalOverrides := make([]*gqlschema.ConfigEntryInput, 0)

	secrets := &coreV1.SecretList{}
	listOpts := secretListOptions()

	if err := ro.k8sClient.List(ro.ctx, secrets, listOpts...); err != nil {
		errMsg := fmt.Sprintf("cannot fetch list of secrets: %s", err)
		return componentsOverrides, globalOverrides, errors.Wrap(err, errMsg)
	}

	for _, secret := range secrets.Items {
		component, global := getComponent(secret.Labels)
		for key, value := range secret.Data {
			if global {
				globalOverrides = append(globalOverrides, &gqlschema.ConfigEntryInput{
					Key:    key,
					Value:  string(value),
					Secret: ptr.Bool(true),
				})
			} else {
				componentsOverrides[component] = append(componentsOverrides[component], &gqlschema.ConfigEntryInput{
					Key:    key,
					Value:  string(value),
					Secret: ptr.Bool(true),
				})
			}
		}
	}

	return componentsOverrides, globalOverrides, nil
}

func (ro *runtimeOverrides) collectFromConfigMaps(planName, overridesVersion, account, subaccount string) (map[string][]*gqlschema.ConfigEntryInput, []*gqlschema.ConfigEntryInput, error) {
	componentsOverrides := make(map[string][]*gqlschema.ConfigEntryInput, 0)
	globalOverrides := make([]*gqlschema.ConfigEntryInput, 0)
	OverrideTypeLabelKeys := []string{overridesPlanLabelPrefix + planName, overridesAccountLabelPrefix + account, overridesSubaccountLabelPrefix + subaccount}

	for _, labelKey := range OverrideTypeLabelKeys {
		ro.log.Debugf("collectFromConfigMaps() labelKey %s on account %s subaccount %s\n", labelKey, account, subaccount)
		configMaps := &coreV1.ConfigMapList{}
		listOpts := configMapListOptions(labelKey, overridesVersion)

		if err := ro.k8sClient.List(ro.ctx, configMaps, listOpts...); err != nil {
			switch labelKey {
			case overridesPlanLabelPrefix + planName:
				return componentsOverrides, globalOverrides, printOverrideError(err)
			default:
				if !apiErrors.IsNotFound(err) {
					return componentsOverrides, globalOverrides, printOverrideError(err)
				}
			}
		}

		for _, cm := range configMaps.Items {
			component, global := getComponent(cm.Labels)
			for key, value := range cm.Data {
				ro.log.Debugf("collectFromConfigMaps() component , global: %s %s overrideType key value : %s %s %s", component, global, labelKey, key, value)
				if global {
					globalOverrides = append(globalOverrides, &gqlschema.ConfigEntryInput{
						Key:   key,
						Value: value,
					})
				} else {
					componentsOverrides[component] = append(componentsOverrides[component], &gqlschema.ConfigEntryInput{
						Key:   key,
						Value: value,
					})
				}
			}
		}
	}
	return componentsOverrides, globalOverrides, nil
}

func printOverrideError(err error) error {
	errMsg := fmt.Sprintf("cannot fetch list of config maps: %s", err)
	return errors.Wrap(err, errMsg)
}

func secretListOptions() []client.ListOption {
	label := map[string]string{
		overridesSecretLabel: "true",
	}

	return []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(label),
	}
}

func configMapListOptions(labelkey string, version string) []client.ListOption {
	var label map[string]string
	versionLabel := overridesVersionLabelPrefix + strings.ToLower(version)

	label = map[string]string{
		labelkey:     "true",
		versionLabel: "true",
	}
	return []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(label),
	}
}

func getComponent(labels map[string]string) (string, bool) {
	for name, value := range labels {
		if name == componentNameLabel {
			return value, false
		}
	}
	return "", true
}

func appendOverrides(input InputAppender, componentsOverrides map[string][]*gqlschema.ConfigEntryInput, globalOverrides []*gqlschema.ConfigEntryInput) {
	for component, overrides := range componentsOverrides {
		input.AppendOverrides(component, overrides)
	}

	if len(globalOverrides) > 0 {
		input.AppendGlobalOverrides(globalOverrides)
	}
}
