package main

type provisionerConfig struct {
	URL          string `envconfig:"default=kcp-provisioner:3000"`
	QueryDumping bool   `envconfig:"default=false"`
}

func main() {
	builder := NewAppBuilder()

	builder.withGardenerClient()
	builder.withBrokerClient()
	builder.withStorage()

	job := builder.Create()

	err := job.Run()

	if err != nil {
		FatalOnError(err)
	}
}
