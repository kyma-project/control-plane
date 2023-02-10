package main

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/setup"

func main() {
	builder := setup.NewAppBuilder()

	builder.withConfig()
	builder.withGardenerClient()
	builder.withBrokerClient()
	builder.withProvisionerClient()
	builder.withStorage()
	builder.withLogger()

	defer builder.Cleanup()

	job := builder.Create()

	err := job.Run()

	if err != nil {
		setup.FatalOnError(err)
	}
}
