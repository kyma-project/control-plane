package main

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/setup"

func main() {
	builder := setup.NewAppBuilder()

	builder.WithConfig()
	builder.WithGardenerClient()
	builder.WithBrokerClient()
	builder.WithProvisionerClient()
	builder.WithStorage()
	builder.WithLogger()

	defer builder.Cleanup()

	job := builder.Create()

	err := job.Run()

	if err != nil {
		setup.FatalOnError(err)
	}
}
