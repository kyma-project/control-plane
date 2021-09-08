package runtime

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/iosafety"

	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	releaseInstallerURLFormat   = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-installer-cluster.yaml"
	onDemandInstallerURLFormat  = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-installer-cluster.yaml"
	releaseComponentsURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
	onDemandComponentsURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-components.yaml"
)

// ComponentsListProvider provides the whole components list for creating a Kyma Runtime
type ComponentsListProvider struct {
	managedRuntimeComponentsYAMLPath string
	httpClient                       HTTPDoer
	components                       map[string][]v1alpha1.KymaComponent
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewComponentsListProvider returns new instance of the ComponentsListProvider
func NewComponentsListProvider(managedRuntimeComponentsYAMLPath string) *ComponentsListProvider {
	return &ComponentsListProvider{
		httpClient:                       http.DefaultClient,
		managedRuntimeComponentsYAMLPath: managedRuntimeComponentsYAMLPath,
		components:                       make(map[string][]v1alpha1.KymaComponent, 0),
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// the file system and merge them together.
func (r *ComponentsListProvider) AllComponents(kymaVersion string) ([]v1alpha1.KymaComponent, error) {
	if cmps, ok := r.components[kymaVersion]; ok {
		return cmps, nil
	}

	kymaComponents, err := r.getKymaComponents(kymaVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while getting Kyma components")
	}

	managedComponents, err := r.getManagedComponents()
	if err != nil {
		return nil, errors.Wrap(err, "while getting managed components")
	}

	allComponents := append(kymaComponents, managedComponents...)

	r.components[kymaVersion] = allComponents
	return allComponents, nil
}

func (r *ComponentsListProvider) getKymaComponents(kymaVersion string) (comp []v1alpha1.KymaComponent, err error) {
	// installerYamlURL := r.getInstallerYamlURL(version)
	if r.isOnDemandRelease(kymaVersion) {
		return r.getKymaComponentsForCustomVersion(kymaVersion)
	}
	return r.getKymaComponentsForReleaseVersion(kymaVersion)


	componentsYamlURL, err := r.getComponentsYamlURL(kymaVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while getting components URL")
	}

	req, err := http.NewRequest(http.MethodGet, componentsYamlURL, nil)
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
			return t.Spec.Components, nil
		}
	}
	return nil, errors.New("installer cr not found")

}

func (r *ComponentsListProvider) getManagedComponents() ([]v1alpha1.KymaComponent, error) {
	yamlFile, err := ioutil.ReadFile(r.managedRuntimeComponentsYAMLPath)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with managed components list")
	}

	var managedList struct {
		Components []v1alpha1.KymaComponent `json:"components"`
	}
	err = yaml.Unmarshal(yamlFile, &managedList)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshaling YAML file with managed components list")
	}
	return managedList.Components, nil
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

func (r *ComponentsListProvider) getInstallerYamlURL(kymaVersion string) string {
	return fmt.Sprintf(releaseInstallerURLFormat, kymaVersion)
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

func (r *ComponentsListProvider) getComponentsYamlURL(kymaVersion string) (string, error) {
	if r.isOnDemandRelease(kymaVersion) {
		return fmt.Sprintf(onDemandInstallerURLFormat, kymaVersion), nil
	}
	return r.determineReleaseComponentsURL(kymaVersion)
}

func (r *ComponentsListProvider) determineReleaseComponentsURL(kymaVersion string) (string, error) {
	kymaMajorVer := r.getMajorVersion(kymaVersion)
	majorVerNum, err := strconv.Atoi(kymaMajorVer)
	if err != nil {
		return "", errors.New("cannot convert Kyma's major version number to int")
	}
	if majorVerNum > 1 {
		return fmt.Sprintf(releaseComponentsURLFormat, kymaVersion), nil
	}
	return fmt.Sprintf(releaseInstallerURLFormat, kymaVersion), nil
}

func (r *ComponentsListProvider) getMajorVersion(version string) string {
	splitVer := strings.Split(version, ".")
	return splitVer[0]
}


func (r *ComponentsListProvider) getKymaComponentsForCustomVersion(kymaVersion string) ([]v1alpha1.KymaComponent, error) {
	cmpsURL := fmt.Sprintf(onDemandInstallerURLFormat, kymaVersion)
	_ = cmpsURL
	return nil, nil
}

func (r *ComponentsListProvider) getKymaComponentsForReleaseVersion(kymaVersion string) ([]v1alpha1.KymaComponent, error) {
	return nil, nil
}
