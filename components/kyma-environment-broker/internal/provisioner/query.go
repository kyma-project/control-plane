package provisioner

import "fmt"

type queryProvider struct{}

func (qp queryProvider) provisionRuntime(config string) string {
	return fmt.Sprintf(`mutation {
	result: provisionRuntime(config: %s) {
		%s
}
}`, config, operationStatusData())
}

func (qp queryProvider) upgradeRuntime(runtimeID string, config string) string {
	return fmt.Sprintf(`mutation {
	result: upgradeRuntime(id: "%s", config: %s) {
		%s
}
}`, runtimeID, config, operationStatusData())
}

func (qp queryProvider) upgradeShoot(runtimeID string, config string) string {
	return fmt.Sprintf(`mutation {
	result: upgradeShoot(id: "%s", config: %s) {
		%s
}
}`, runtimeID, config, operationStatusData())
}

func (qp queryProvider) deprovisionRuntime(runtimeID string) string {
	return fmt.Sprintf(`mutation {
	result: deprovisionRuntime(id: "%s")
}`, runtimeID)
}

func (qp queryProvider) reconnectRuntimeAgent(runtimeID string) string {
	return fmt.Sprintf(`mutation {
	result: reconnectRuntimeAgent(id: "%s")
}`, runtimeID)
}

func (qp queryProvider) runtimeStatus(runtimeID string) string {
	return fmt.Sprintf(`query {
	result: runtimeStatus(id: "%s") {
	%s
	}
}`, runtimeID, runtimeStatusData())
}

func (qp queryProvider) runtimeOperationStatus(operationID string) string {
	return fmt.Sprintf(`query {
	result: runtimeOperationStatus(id: "%s") {
	%s
	}
}`, operationID, operationStatusData())
}

func runtimeStatusData() string {
	return fmt.Sprintf(`lastOperationStatus { operation state message }
			runtimeConnectionStatus { status }
			runtimeConfiguration {
				kubeconfig
				clusterConfig {
					%s
				}
				kymaConfig { version }
			}`, clusterConfig())
}

func clusterConfig() string {
	return fmt.Sprintf(`
		name
		kubernetesVersion
		volumeSizeGB
		diskType
		machineType
		region
		provider
		seed
		targetSecret
		diskType
		workerCidr
		autoScalerMin
		autoScalerMax
		maxSurge
		maxUnavailable
		providerSpecificConfig {
			%s
		}
`, providerSpecificConfig())
}

func providerSpecificConfig() string {
	return fmt.Sprint(`
		... on GCPProviderConfig {
			zones
		}
		... on AzureProviderConfig {
			vnetCidr
		}
		... on AWSProviderConfig {
			zones {
                  ... on AWSZone {
                    name
                    publicCidr
                    workerCidr
                    internalCidr
                  }
			}
			vpcCidr
		}
	`)
}

func operationStatusData() string {
	return `id
			operation
			state
			message
			runtimeID`
}
