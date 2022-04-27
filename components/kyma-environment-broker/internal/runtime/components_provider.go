package runtime

import (
	"net/http"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KymaComponent struct{}

// ComponentsProvider provides the list of required and additional components for creating a Kyma Runtime
type ComponentsProvider struct {
	httpClient HTTPDoer
	k8sClient  client.Client
	components map[string]map[string][]KymaComponent // runtimeversion -> plan -> components
	mu         sync.Mutex
}

// NewComponentsProvider returns new instance of the ComponentsProvider
func NewComponentsProvider(k8sClient client.Client) *ComponentsProvider {
	return &ComponentsProvider{
		httpClient: http.DefaultClient,
		k8sClient:  k8sClient,
		components: make(map[string]map[string][]KymaComponent, 0),
	}
}

// AllComponents returns all components for Kyma Runtime. It fetches always the
// Kyma open-source components from the given url and management components from
// ConfigMaps and merge them together
func (p *ComponentsProvider) AllComponents(kymaVersion internal.RuntimeVersionData) ([]KymaComponent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
}
