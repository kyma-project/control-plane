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
)

const (
	releaseComponentsURLFormat  = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
	onDemandComponentsURLFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-components.yaml"
)

// ComponentsProvider provides the list of required and additional components for creating a Kyma Runtime
type ComponentsProvider struct {
	mu         sync.Mutex
	httpClient HTTPDoer
}

// NewComponentsProvider returns new instance of the ComponentsProvider
func NewComponentsProvider() *ComponentsProvider {
	return &ComponentsProvider{
		httpClient: http.DefaultClient,
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// ConfigMaps and merge them together
func (p *ComponentsProvider) AllComponents(kymaVersion internal.RuntimeVersionData, config *internal.ConfigForPlan) ([]internal.KymaComponent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	kymaComponents, err := p.requiredComponents(kymaVersion)
	if err != nil {
		return nil, fmt.Errorf("while getting Kyma components: %w", err)
	}

	additionalComponents := config.AdditionalComponents
	if err != nil {
		return nil, fmt.Errorf("while getting additional components: %w", err)
	}

	allComponents := append(kymaComponents, additionalComponents...)

	return allComponents, nil

}

func (p *ComponentsProvider) requiredComponents(kymaVersion internal.RuntimeVersionData) ([]internal.KymaComponent,
	error) {
	return p.getComponentsFromComponentsYaml(kymaVersion.Version)

}

func (p *ComponentsProvider) getComponentsFromComponentsYaml(version string) ([]internal.KymaComponent, error) {
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
		DefaultNamespace string                   `yaml:"defaultNamespace"`
		Prerequisites    []internal.KymaComponent `yaml:"prerequisites"`
		Components       []internal.KymaComponent `yaml:"components"`
	}

	decoder := yaml.NewDecoder(resp.Body)

	var kymaCmps kymaComponents
	if err = decoder.Decode(&kymaCmps); err != nil {
		return nil, fmt.Errorf("while decoding response body: %w", err)
	}

	requiredComponents := make([]internal.KymaComponent, 0)
	requiredComponents = append(requiredComponents, kymaCmps.Prerequisites...)
	requiredComponents = append(requiredComponents, kymaCmps.Components...)

	for i, cmp := range requiredComponents {
		if cmp.Namespace == "" {
			requiredComponents[i].Namespace = kymaCmps.DefaultNamespace
		}
	}

	return requiredComponents, nil
}

func (p *ComponentsProvider) getComponentsYamlURL(kymaVersion string) string {
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
func (p *ComponentsProvider) isOnDemandRelease(version string) bool {
	isOnDemandVersion := strings.HasPrefix(version, "PR-") ||
		strings.HasPrefix(version, "main-")
	return isOnDemandVersion
}

func (p *ComponentsProvider) checkStatusCode(resp *http.Response) error {
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
