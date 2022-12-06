package runtimeoverrides

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace                   = "kcp-system"
	componentNameLabel          = "component"
	overridesVersionLabelPrefix = "overrides-version-"
	overridesPlanLabelPrefix    = "overrides-plan-"
	overridesSecretLabel        = "runtime-override"
)

type InputAppender interface {
	AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator
	AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator
}

type runtimeOverrides struct {
	ctx       context.Context
	k8sClient client.Client
}

func NewRuntimeOverrides(ctx context.Context, cli client.Client) *runtimeOverrides {
	return &runtimeOverrides{
		ctx:       ctx,
		k8sClient: cli,
	}
}

func (ro *runtimeOverrides) Append(input InputAppender, planName, overridesVersion string) error {
	{
		componentsOverrides, globalOverrides, err := ro.collectFromSecrets()
		if err != nil {
			return fmt.Errorf("cannot collect overrides from secrets: %w", err)
		}

		appendOverrides(input, componentsOverrides, globalOverrides)
	}

	{
		componentsOverrides, globalOverrides, err := ro.collectFromConfigMaps(planName, overridesVersion)
		if err != nil {
			return fmt.Errorf("cannot collect overrides from config maps: %w", err)
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
		return componentsOverrides, globalOverrides, fmt.Errorf("cannot fetch list of secrets: %w", err)
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

func (ro *runtimeOverrides) collectFromConfigMaps(planName, overridesVersion string) (map[string][]*gqlschema.ConfigEntryInput, []*gqlschema.ConfigEntryInput, error) {
	componentsOverrides := make(map[string][]*gqlschema.ConfigEntryInput, 0)
	globalOverrides := make([]*gqlschema.ConfigEntryInput, 0)

	configMaps := &coreV1.ConfigMapList{}
	listOpts := configMapListOptions(planName, overridesVersion)

	if err := ro.k8sClient.List(ro.ctx, configMaps, listOpts...); err != nil {
		return componentsOverrides, globalOverrides, fmt.Errorf("cannot fetch list of config maps: %w", err)
	}

	for _, cm := range configMaps.Items {
		component, global := getComponent(cm.Labels)
		for key, value := range cm.Data {
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

	return componentsOverrides, globalOverrides, nil
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

func configMapListOptions(plan string, version string) []client.ListOption {
	planLabel := overridesPlanLabelPrefix + plan
	versionLabel := overridesVersionLabelPrefix + strings.ToLower(version)

	label := map[string]string{
		planLabel:    "true",
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
