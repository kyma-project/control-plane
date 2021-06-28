package provisioner

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type runtime struct {
	runtimeInput schema.ProvisionRuntimeInput
}

type FakeClient struct {
	mu            sync.Mutex
	graphqlizer   Graphqlizer

	runtimes      []runtime
	upgrades      map[string]schema.UpgradeRuntimeInput
	shootUpgrades map[string]schema.UpgradeShootInput
	operations    map[string]schema.OperationStatus
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		graphqlizer: Graphqlizer{},
		runtimes:      []runtime{},
		operations:    make(map[string]schema.OperationStatus),
		upgrades:      make(map[string]schema.UpgradeRuntimeInput),
		shootUpgrades: make(map[string]schema.UpgradeShootInput),
	}
}

func (c *FakeClient) GetProvisionRuntimeInput(index int) schema.ProvisionRuntimeInput {
	c.mu.Lock()
	defer c.mu.Unlock()

	r := c.runtimes[index]
	return r.runtimeInput
}

func (c *FakeClient) FinishProvisionerOperation(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	op := c.operations[id]
	op.State = schema.OperationStateSucceeded
	c.operations[id] = op
}

func (c *FakeClient) FindOperationByRuntimeIDAndType(runtimeID string, operationType schema.OperationType) schema.OperationStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, status := range c.operations {
		if *status.RuntimeID == runtimeID && status.Operation == operationType {
			return status
		}
	}
	for _, status := range c.operations {
		if *status.RuntimeID == runtimeID && status.Operation == operationType {
			return status
		}
	}
	return schema.OperationStatus{}
}

func (c *FakeClient) SetOperation(id string, operation schema.OperationStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.operations[id] = operation
}

// Provisioner Client methods

func (c *FakeClient) ProvisionRuntime(accountID, subAccountID string, config schema.ProvisionRuntimeInput) (schema.OperationStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rid := uuid.New().String()
	opId := uuid.New().String()
	c.runtimes = append(c.runtimes, runtime{
		runtimeInput: config,
	})
	c.operations[opId] = schema.OperationStatus{
		ID:        &opId,
		RuntimeID: &rid,
		Operation: schema.OperationTypeProvision,
		State:     schema.OperationStateInProgress,
	}
	return schema.OperationStatus{
		RuntimeID: &rid,
		ID:        &opId,
	}, nil
}

func (c *FakeClient) DeprovisionRuntime(accountID, runtimeID string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	opId := uuid.New().String()

	c.operations[opId] = schema.OperationStatus{
		ID:        &opId,
		Operation: schema.OperationTypeDeprovision,
		State:     schema.OperationStateInProgress,
		RuntimeID: &runtimeID,
	}

	return opId, nil
}

func (c *FakeClient) ReconnectRuntimeAgent(accountID, runtimeID string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *FakeClient) RuntimeOperationStatus(accountID, operationID string) (schema.OperationStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	o, found := c.operations[operationID]
	if !found {
		return schema.OperationStatus{}, fmt.Errorf("operation not found")
	}
	return o, nil
}

func (c *FakeClient) RuntimeStatus(accountID, runtimeID string) (schema.RuntimeStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return schema.RuntimeStatus{
		RuntimeConfiguration: &schema.RuntimeConfig{
			ClusterConfig: &schema.GardenerConfig{
				Name:   ptr.String("fake-name"),
				Region: ptr.String("fake-region"),
				Seed:   ptr.String("fake-seed"),
			},
		},
	}, nil
}

func (c *FakeClient) UpgradeRuntime(accountID, runtimeID string, config schema.UpgradeRuntimeInput) (schema.OperationStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	opId := uuid.New().String()
	c.operations[opId] = schema.OperationStatus{
		ID:        &opId,
		RuntimeID: &runtimeID,
		Operation: schema.OperationTypeUpgrade,
		State:     schema.OperationStateInProgress,
	}
	c.upgrades[runtimeID] = config
	return schema.OperationStatus{
		RuntimeID: &runtimeID,
		ID:        &opId,
	}, nil
}

func (c *FakeClient) UpgradeShoot(accountID, runtimeID string, config schema.UpgradeShootInput) (schema.OperationStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	upgradeShootIptGQL, _ := c.graphqlizer.UpgradeShootInputToGraphQL(config)
	fmt.Println(upgradeShootIptGQL)

	opId := uuid.New().String()
	c.operations[opId] = schema.OperationStatus{
		ID:        &opId,
		RuntimeID: &runtimeID,
		Operation: schema.OperationTypeUpgradeShoot,
		State:     schema.OperationStateInProgress,
	}
	c.shootUpgrades[runtimeID] = config
	return schema.OperationStatus{
		RuntimeID: &runtimeID,
		ID:        &opId,
	}, nil
}

func (c *FakeClient) IsRuntimeUpgraded(runtimeID string, version string) bool {
	input, found := c.upgrades[runtimeID]
	if found && version != "" && input.KymaConfig != nil {
		return input.KymaConfig.Version == version
	}

	return found
}

func (c *FakeClient) IsShootUpgraded(runtimeID string) bool {
	_, found := c.shootUpgrades[runtimeID]
	return found
}

func (c *FakeClient) LastShootUpgrade(runtimeID string) (schema.UpgradeShootInput, bool) {
	input, found := c.shootUpgrades[runtimeID]
	return input, found
}
