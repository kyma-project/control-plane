package director

type Config struct {
	URL                  string `envconfig:"default=http://compass-director.compass-system.svc.cluster.local:3000/graphql"`
	Namespace            string `envconfig:"default=kcp-system"`
	SkipCertVerification bool   `envconfig:"default=false"`
	OauthTokenURL        string
	OauthClientID        string
	OauthClientSecret    string
	OauthScope           string
}
