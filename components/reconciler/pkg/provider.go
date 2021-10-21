package mothership

import "net/url"

//go:generate mockgen -destination=automock/provider.go -package=automock . URLProvider

const (
	EndpointReconcile string = "reconcile"
)

type URLProvider interface {
	Provide(endpoint string, queryParams map[string]string) url.URL
}

type urlProvider struct {
	mothershipURL url.URL
}

func (p urlProvider) Provide(endpoint string, queryParams map[string]string) url.URL {

	query := url.Values{}
	for k, v := range queryParams {
		query.Add(k, v)
	}

	rawQuery := query.Encode()

	result := url.URL(p.mothershipURL)
	result.Path = endpoint
	result.RawQuery = rawQuery

	return result
}
