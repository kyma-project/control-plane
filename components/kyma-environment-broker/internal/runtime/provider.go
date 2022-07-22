package runtime

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/iosafety"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	releaseInstallerURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-installer-cluster.yaml"
	onDemandInstallerURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-installer-cluster.yaml"
)

// ComponentsListProvider provides the whole components list for creating a Kyma Runtime
type ComponentsListProvider struct {
	managedRuntimeComponentsYAMLPath       string
	newAdditionalRuntimeComponentsYAMLPath string
	httpClient                             HTTPDoer
	components                             map[string][]internal.KymaComponent
	mu                                     sync.Mutex
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewComponentsListProvider returns new instance of the ComponentsListProvider
func NewComponentsListProvider(managedRuntimeComponentsYAMLPath, newAdditionalRuntimeComponentsYAMLPath string) *ComponentsListProvider {
	return &ComponentsListProvider{
		httpClient:                             http.DefaultClient,
		managedRuntimeComponentsYAMLPath:       managedRuntimeComponentsYAMLPath,
		newAdditionalRuntimeComponentsYAMLPath: newAdditionalRuntimeComponentsYAMLPath,
		components:                             make(map[string][]internal.KymaComponent, 0),
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// the file system and merge them together.
func (r *ComponentsListProvider) AllComponents(kymaVersion internal.RuntimeVersionData, planName string) ([]internal.KymaComponent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cmps, ok := r.components[kymaVersion.Version]; ok {
		return cmps, nil
	}

	kymaComponents, err := r.getKymaComponents(kymaVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while getting Kyma components")
	}

	additionalComponents, err := r.getAdditionalComponents(kymaVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while getting additional components")
	}

	allComponents := append(kymaComponents, additionalComponents...)

	r.components[kymaVersion.Version] = allComponents
	return allComponents, nil
}

func (r *ComponentsListProvider) getKymaComponents(kymaVersion internal.RuntimeVersionData) (comp []internal.KymaComponent, err error) {
	if kymaVersion.MajorVersion > 1 {
		return r.getComponentsFromComponentsYaml(kymaVersion.Version)
	}
	return r.getComponentsFromInstallerYaml(kymaVersion.Version)
}

func (r *ComponentsListProvider) getAdditionalComponents(kymaVersion internal.RuntimeVersionData) ([]internal.KymaComponent, error) {
	if kymaVersion.MajorVersion > 1 {
		return r.getAdditionalComponentsForNewKyma()
	}
	return r.getAdditionalComponentsForKyma()
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

type kymaComponents struct {
	DefaultNamespace string                   `yaml:"defaultNamespace"`
	Prerequisites    []internal.KymaComponent `yaml:"prerequisites"`
	Components       []internal.KymaComponent `yaml:"components"`
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

func (r *ComponentsListProvider) getInstallerYamlURL(kymaVersion string) string {
	if r.isOnDemandRelease(kymaVersion) {
		return fmt.Sprintf(onDemandInstallerURLFormat, kymaVersion)
	}
	return fmt.Sprintf(releaseInstallerURLFormat, kymaVersion)
}

func (r *ComponentsListProvider) getComponentsYamlURL(kymaVersion string) string {
	if r.isOnDemandRelease(kymaVersion) {
		return fmt.Sprintf(onDemandComponentsURLFormat, kymaVersion)
	}
	return fmt.Sprintf(releaseComponentsURLFormat, kymaVersion)
}

func (r *ComponentsListProvider) getComponentsFromComponentsYaml(kymaVersion string) ([]internal.KymaComponent, error) {
	yamlURL := r.getComponentsYamlURL(kymaVersion)

	req, err := http.NewRequest(http.MethodGet, yamlURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating http request")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, kebError.AsTemporaryError(err, "while making request for Kyma components list")
	}
	defer func() {
		if drainErr := iosafety.DrainReader(resp.Body); drainErr != nil {
			err = multierror.Append(err, errors.Wrap(drainErr, "while trying to drain body reader"))
		}

		if closeErr := resp.Body.Close(); closeErr != nil {
			err = multierror.Append(err, errors.Wrap(closeErr, "while trying to close body reader"))
		}
	}()

	if err = r.checkStatusCode(resp); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(resp.Body)

	var kymaCmps kymaComponents
	if err = dec.Decode(&kymaCmps); err != nil {
		return nil, err
	}

	allKymaComponents := make([]internal.KymaComponent, 0)
	allKymaComponents = append(allKymaComponents, kymaCmps.Prerequisites...)
	allKymaComponents = append(allKymaComponents, kymaCmps.Components...)

	for i, cmp := range allKymaComponents {
		if cmp.Namespace == "" {
			allKymaComponents[i].Namespace = kymaCmps.DefaultNamespace
		}
	}

	return allKymaComponents, nil
}

func (r *ComponentsListProvider) getComponentsFromInstallerYaml(kymaVersion string) ([]internal.KymaComponent, error) {
	yamlURL := r.getInstallerYamlURL(kymaVersion)

	req, err := http.NewRequest(http.MethodGet, yamlURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating http request")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, kebError.AsTemporaryError(err, "while making request for Kyma components list")
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
		return nil, err
	}

	dec := yaml.NewDecoder(resp.Body)

	var t Installation
	for dec.Decode(&t) == nil {
		if t.Kind == "Installation" {
			components := make([]internal.KymaComponent, len(t.Spec.Components))
			for i, cmp := range t.Spec.Components {
				components[i] = r.v1alpha1ToKymaComponent(cmp)
			}
			return components, nil
		}
	}
	return nil, errors.New("installer cr not found")
}

func (r *ComponentsListProvider) getAdditionalComponentsForNewKyma() ([]internal.KymaComponent, error) {
	yamlContents, err := r.readYAML(r.newAdditionalRuntimeComponentsYAMLPath)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with additional components for new Kyma")
	}
	return r.getComponentsFromYAML(yamlContents)
}

func (r *ComponentsListProvider) getAdditionalComponentsForKyma() ([]internal.KymaComponent, error) {
	yamlContents, err := r.readYAML(r.managedRuntimeComponentsYAMLPath)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with additional components")
	}
	return r.getComponentsFromYAML(yamlContents)
}

func (r *ComponentsListProvider) readYAML(yamlFilePath string) ([]byte, error) {
	yamlContents, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "while trying to read YAML file")
	}
	return yamlContents, nil
}

func (r *ComponentsListProvider) getComponentsFromYAML(yamlFileContents []byte) ([]internal.KymaComponent, error) {
	var components struct {
		Components []internal.KymaComponent `json:"components"`
	}
	err := yaml.Unmarshal(yamlFileContents, &components)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshalling YAML file with additional components")
	}
	return components.Components, nil
}

func (r *ComponentsListProvider) v1alpha1ToKymaComponent(cmp v1alpha1.KymaComponent) internal.KymaComponent {
	var source *internal.ComponentSource
	if cmp.Source != nil {
		source.URL = cmp.Source.URL
	}

	return internal.KymaComponent{
		Name:        cmp.Name,
		ReleaseName: cmp.ReleaseName,
		Namespace:   cmp.Namespace,
		Source:      source,
	}
}
