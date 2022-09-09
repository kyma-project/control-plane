package dashboard

type Config struct {
	LandscapeURL string `envconfig:"default=https://dashboard.kyma.cloud.sap"`
}
