package director

type Config struct {
	URL               string `envconfig:"default=http://compass-director.compass-system.svc.cluster.local:3000/graphql"`
	OauthTokenURL     string `envconfig:"default=https://oauth.domain.com/oauth/token"`
	OauthClientID     string `envcondif:"default=directorId"`
	OauthClientSecret string `envconfig:"default=directorSecret"`
	OauthScope        string `envconfig:"default=runtime:read runtime:write"`
}
