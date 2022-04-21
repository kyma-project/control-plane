package dashboard

type Config struct {
	Enabled      bool   `envconfig:"default=false"`
	LandscapeURL string `envconfig:"default=https://dashboard.kyma.cloud.sap"`
}
