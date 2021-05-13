package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

// The providerKey type is no exported to prevent collisions with context keys
// defined in other packages.
type providerKey int

const (
	// requestRegionKey is the context key for the region from the request path.
	requestProviderKey providerKey = iota + 1
)

func AddProviderToContext() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			vars := mux.Vars(req)
			region, found := vars["region"]
			provider := internal.UnknownProvider
			if found {
				provider = platformProvider(region)
			}

			newCtx := context.WithValue(req.Context(), requestProviderKey, provider)
			next.ServeHTTP(w, req.WithContext(newCtx))
		})
	}
}

// ProviderFromContext returns request provider associated with the context if possible.
func ProviderFromContext(ctx context.Context) (internal.CloudProvider, bool) {
	provider, ok := ctx.Value(requestProviderKey).(internal.CloudProvider)
	return provider, ok
}

var platformRegionProviderRE = regexp.MustCompile("[0-9]")

func platformProvider(region string) internal.CloudProvider {
	if region == "" {
		return internal.UnknownProvider
	}
	digit := platformRegionProviderRE.FindString(region)
	switch digit {
	case "1":
		return internal.AWS
	case "2":
		return internal.Azure
	default:
		return internal.UnknownProvider
	}
}
