package runtime

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/iosafety"

	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	releaseInstallerURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-installer-cluster.yaml"
	onDemandInstallerURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-installer-cluster.yaml"

	releaseComponentsURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
	onDemandComponentsURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-components.yaml"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ListDecider interface {
	IsNewComponentList(string) (bool, error)
}

// Installation represents the installer CR.
// It is copied because using directly the installer CR
// with such fields:
//
// 	metav1.TypeMeta   `json:",inline"`
//	metav1.ObjectMeta `json:"metadata,omitempty"`
//
// is not working with "gopkg.in/yaml.v2" stream decoder.
// On the other hand "sigs.k8s.io/yaml" does not support
// stream decoding.
type Installation struct {
	Kind string                    `json:"kind"`
	Spec v1alpha1.InstallationSpec `json:"spec"`
}

// ComponentsListProvider provides the whole components list for creating a Kyma Runtime
type ComponentsListProvider struct {
	listDecider                      ListDecider
	managedRuntimeComponentsYAMLPath string
	httpClient                       HTTPDoer
	components                       map[string]ComponentListData
}

// NewComponentsListProvider returns new instance of the ComponentsListProvider
func NewComponentsListProvider(listDecider ListDecider, managedRuntimeComponentsYAMLPath string) *ComponentsListProvider {
	return &ComponentsListProvider{
		listDecider:                      listDecider,
		httpClient:                       http.DefaultClient,
		managedRuntimeComponentsYAMLPath: managedRuntimeComponentsYAMLPath,
		components:                       make(map[string]ComponentListData, 0),
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// the file system and merge them together.
func (r *ComponentsListProvider) AllComponents(kymaVersion string) (ComponentListData, error) {
	if cmps, ok := r.components[kymaVersion]; ok {
		return cmps, nil
	}

	var components ComponentListData

	// Read Kyma installer yaml (url)
	components, err := r.getOpenSourceKymaComponents(kymaVersion)
	if err != nil {
		return components, errors.Wrap(err, "while getting open source kyma components")
	}

	// Read mounted config (path)
	managedRuntimeComponents, err := r.getManagedRuntimeComponents()
	if err != nil {
		return components, errors.Wrap(err, "while getting managed runtime components list")
	}

	// Add managed components from config to open source Kyma components
	components.Components = append(components.Components, managedRuntimeComponents...)

	// Fill in the missing namespace fields
	for idx := range components.Components {
		if components.Components[idx].Namespace == "" {
			components.Components[idx].Namespace = components.DefaultNamespace
		}
	}
	for idx := range components.Prerequisites {
		if components.Prerequisites[idx].Namespace == "" {
			components.Prerequisites[idx].Namespace = components.DefaultNamespace
		}
	}

	r.components[kymaVersion] = components
	return components, nil
}

func (r *ComponentsListProvider) getOpenSourceKymaComponents(version string) (components ComponentListData, err error) {
	// check if a demanded version requires new component list (for new parallel installation)
	// TODO: when only new parallel installation will be used, the condition could be removed, only new list will be used
	newList, err := r.listDecider.IsNewComponentList(version)
	if err != nil {
		return components, errors.Wrap(err, "while specifying old/new component list should be used")
	}

	componentsYamlURL := r.getComponentsYamlURL(newList, version)

	req, err := http.NewRequest(http.MethodGet, componentsYamlURL, nil)
	if err != nil {
		return components, errors.Wrap(err, "while creating http request")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return components, kebError.AsTemporaryError(err, "while making request for Kyma components list")
	}
	defer func() {
		if drainErr := iosafety.DrainReader(resp.Body); drainErr != nil {
			err = multierror.Append(err, errors.Wrap(drainErr, "while trying to drain body reader"))
		}

		if closeErr := resp.Body.Close(); closeErr != nil {
			err = multierror.Append(err, errors.Wrap(closeErr, "while trying to close body reader"))
		}
	}()

	if err := r.checkStatusCode(resp); err != nil {
		return components, err
	}

	if newList {
		err = yaml.NewDecoder(resp.Body).Decode(&components)
		if err != nil {
			return components, errors.Wrap(err, "while decoding Kyma components list")
		}
	} else {
		return r.transformInstallationToComponentList(resp)
	}

	return components, nil
}

func (r *ComponentsListProvider) getManagedRuntimeComponents() ([]ComponentDefinition, error) {
	yamlFile, err := ioutil.ReadFile(r.managedRuntimeComponentsYAMLPath)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with managed components list")
	}

	var managedList struct {
		Components []ComponentDefinition `json:"components"`
	}
	err = yaml.Unmarshal(yamlFile, &managedList)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshalling YAML file with managed components list")
	}

	return managedList.Components, nil
}

func (r *ComponentsListProvider) checkStatusCode(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// limited buff to ready only ~4kb, so big response will not blowup our component
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		body = []byte(fmt.Sprintf("cannot read body, got error: %s", err))
	}
	msg := fmt.Sprintf("while checking response status code for Kyma components list: "+
		"got unexpected status code, want %d, got %d, url: %s, body: %s",
		http.StatusOK, resp.StatusCode, resp.Request.URL.String(), body)

	switch {
	case resp.StatusCode == http.StatusRequestTimeout:
		return kebError.NewTemporaryError(msg)
	case resp.StatusCode >= http.StatusInternalServerError:
		return kebError.NewTemporaryError(msg)
	default:
		return errors.New(msg)
	}
}

func (r *ComponentsListProvider) getComponentsYamlURL(new bool, kymaVersion string) string {
	// default value of URL is to artefact with Installation resource
	// if new installation will be used then list component could be find in kyma-component.yaml file
	// TODO: when only new parallel installation will be used, only URL to kyma-component.yaml will be valid
	var (
		onDemand = onDemandInstallerURLFormat
		release  = releaseInstallerURLFormat
	)

	if new {
		onDemand = onDemandComponentsURLFormat
		release = releaseComponentsURLFormat
	}

	if r.isOnDemandRelease(kymaVersion) {
		return fmt.Sprintf(onDemand, kymaVersion)
	}
	return fmt.Sprintf(release, kymaVersion)
}

// isOnDemandRelease returns true if the version is recognized as on-demand.
//
// Detection rules:
//   For pull requests: PR-<number>
//   For changes to the main branch: main-<commit_sha>
//
// source: https://github.com/kyma-project/test-infra/blob/main/docs/prow/prow-architecture.md#generate-development-artifacts
func (r *ComponentsListProvider) isOnDemandRelease(version string) bool {
	isOnDemandVersion := strings.HasPrefix(version, "PR-") ||
		strings.HasPrefix(version, "main-")
	return isOnDemandVersion
}

func (r *ComponentsListProvider) transformInstallationToComponentList(response *http.Response) (ComponentListData, error) {
	var components ComponentListData
	dec := yaml.NewDecoder(response.Body)

	var t Installation
	for dec.Decode(&t) == nil {
		if t.Kind != "Installation" {
			continue
		}
		for _, c := range t.Spec.Components {
			sourceURL := &ComponentSource{}
			if c.Source != nil {
				sourceURL.URL = c.Source.URL
			} else {
				sourceURL = nil
			}
			components.Components = append(components.Components, ComponentDefinition{
				Name:      c.Name,
				Namespace: c.Namespace,
				Source:    sourceURL,
			})
		}
		components.DefaultNamespace = "kyma-system"
		return components, nil
	}

	return components, errors.New("installer cr not found")
}
