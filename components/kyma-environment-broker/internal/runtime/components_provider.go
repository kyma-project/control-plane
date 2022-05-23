package runtime

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/iosafety"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	releaseComponentsURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
	onDemandComponentsURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-components.yaml"
)

type KymaComponent struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source,omitempty"`
}

type key struct {
	runtimeVersion, plan string
}

type RequiredComponentsProvider interface {
	RequiredComponents(kymaVersion internal.RuntimeVersionData) ([]KymaComponent, error)
}

type AdditionalComponentsProvider interface {
	AdditionalComponents(kymaVersion internal.RuntimeVersionData, plan string) ([]KymaComponent, error)
}

type defaultRequiredComponentsProvider struct {
	httpClient HTTPDoer
}

type defaultAdditionalComponentsProvider struct {
	k8sClient client.Client
}

// ComponentsProvider provides the list of required and additional components for creating a Kyma Runtime
type ComponentsProvider struct {
	requiredComponentsProvider   RequiredComponentsProvider
	additionalComponentsProvider AdditionalComponentsProvider
	components                   map[key][]KymaComponent // runtimeversion -> plan -> components
	mu                           sync.Mutex
}

// NewComponentsProvider returns new instance of the ComponentsProvider
func NewComponentsProvider(k8sClient client.Client) *ComponentsProvider {
	return &ComponentsProvider{
		requiredComponentsProvider:   &defaultRequiredComponentsProvider{httpClient: http.DefaultClient},
		additionalComponentsProvider: &defaultAdditionalComponentsProvider{k8sClient: k8sClient},
		components:                   make(map[key][]KymaComponent, 0),
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// ConfigMaps and merge them together
func (p *ComponentsProvider) AllComponents(kymaVersion internal.RuntimeVersionData, plan string) ([]KymaComponent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cmps, ok := p.components[key{kymaVersion.Version, plan}]; ok {
		return cmps, nil
	}

	kymaComponents, err := p.requiredComponentsProvider.RequiredComponents(kymaVersion)
	if err != nil {
		return nil, fmt.Errorf("while gettiny Kyma components: %w", err)
	}

}

func (p *defaultRequiredComponentsProvider) RequiredComponents(kymaVersion internal.RuntimeVersionData) ([]KymaComponent, error) {
	if kymaVersion.MajorVersion == 2 {
		return p.getComponentsFromComponentsYaml(kymaVersion.Version)
	}
	return nil, errors.New("unsupported Kyma version")
}

func (p *defaultRequiredComponentsProvider) getComponentsFromComponentsYaml(version string) ([]KymaComponent, error) {
	yamlURL := p.getComponentsYamlURL(version)

	req, err := http.NewRequest(http.MethodGet, yamlURL, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating HTTP request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, kebError.AsTemporaryError(err, "while making HTTP request for Kyma components list")
	}
	defer func() {
		if drainErr := iosafety.DrainReader(resp.Body); drainErr != nil {
			err = fmt.Errorf("while trying to drain body reader: %w", err)
		}
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("while trying to close response body reader: %w", err)
		}
	}()

	if err = p.checkStatusCode(resp); err != nil {
		return nil, err
	}

	type kymaComponents struct {
		DefaultNamespace string          `yaml:"defaultNamespace"`
		Prerequisites    []KymaComponent `yaml:"prerequisites"`
		Components       []KymaComponent `yaml:"components"`
	}

	decoder := yaml.NewDecoder(resp.Body)

	var kymaCmps kymaComponents
	if err = decoder.Decode(&kymaCmps); err != nil {
		return nil, fmt.Errorf("while decoding response body: %w", err)
	}

	requiredComponents := make([]KymaComponent, 0)
	requiredComponents = append(requiredComponents, kymaCmps.Prerequisites...)
	requiredComponents = append(requiredComponents, kymaCmps.Components...)

	for i, cmp := range requiredComponents {
		if cmp.Namespace == "" {
			requiredComponents[i].Namespace = kymaCmps.DefaultNamespace
		}
	}

	return requiredComponents, nil
}

func (p *defaultRequiredComponentsProvider) getComponentsYamlURL(kymaVersion string) string {
	if p.isOnDemandRelease(kymaVersion) {
		return fmt.Sprintf(onDemandComponentsURLFormat, kymaVersion)
	}
	return fmt.Sprintf(releaseComponentsURLFormat, kymaVersion)
}

// isOnDemandRelease returns true if the version is recognized as on-demand.
//
// Detection rules:
//   For pull requests: PR-<number>
//   For changes to the main branch: main-<commit_sha>
//
// source: https://github.com/kyma-project/test-infra/blob/main/docs/prow/prow-architecture.md#generate-development-artifacts
func (p *defaultRequiredComponentsProvider) isOnDemandRelease(version string) bool {
	isOnDemandVersion := strings.HasPrefix(version, "PR-") ||
		strings.HasPrefix(version, "main-")
	return isOnDemandVersion
}

func (p *defaultRequiredComponentsProvider) checkStatusCode(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// limited buff to ready only ~4kb, so big response will not blowup our component
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
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

func (p *defaultAdditionalComponentsProvider) AdditionalComponents(kymaVersion internal.RuntimeVersionData, plan string) ([]KymaComponent, error) {

}

func (p *ComponentsProvider) SetRequiredComponentsProvider(provider RequiredComponentsProvider) {
	p.requiredComponentsProvider = provider
}

func (p *ComponentsProvider) SetAdditionalComponentsProvider(provider AdditionalComponentsProvider) {
	p.additionalComponentsProvider = provider
}
