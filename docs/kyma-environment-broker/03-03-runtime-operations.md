# Runtime operations

Kyma Environment Broker allows you to configure operations that you can run on a Runtime. Each operation consists of several steps and each step is represented by a separate file. As every step can be re-launched multiple times, for each step, you should determine a behavior in case of a processing failure. It can either:

- return an error, which interrupts the entire process, or
- repeat the entire operation after the specified period.

> **NOTE:** It's important to set lower timeouts for the Kyma installation in the Runtime Provisioner.

## Provisioning

Each provisioning step is responsible for a separate part of preparing Runtime parameters. For example, in a step you can provide tokens, credentials, or URLs to integrate Kyma Runtime with external systems. All data collected in provisioning steps are used in the step called [`create_cluster_configuration`](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/internal/process/provisioning/create_cluster_configuration.go) which transforms the data into a request input. The request is sent to the Runtime Provisioner component which provisions a Runtime.
The provisioning process contains the following steps:

| Stage          | Step                               | Domain                   | Description                                                                                                                                 | Owner           |
|----------------|------------------------------------|--------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|-----------------|
| start          | Starting                           | Provisioning             | Changes the state from `pending` to `in progress` if there is no other operation in progress.                                               | Team Gopher     |
| create_runtime | Provision_Initialization           | Provisioning             | Starts the provisioning process.                                                                                                            | Team Gopher     |
| create_runtime | Resolve_Target_Secret              | Hyperscaler Account Pool | Provides the name of the Gardener Secret that contains the Hypescaler account credentials used during cluster provisioning.                 | Team Framefrog  |
| create_runtime | AVS_Create_Internal_Eval_Step      | AvS                      | Sets up internal monitoring of Kyma Runtime.                                                                                                | Team Gopher     |
| create_runtime | EDP_Registration                   | Event Data Platform      | Registers an SKR on the Event Data Platform with the necessary parameters. **Note that this step is not mandatory and you can disable it.** | Team Gopher     |
| create_runtime | Overrides_From_Secrets_And_Config_Step | Kyma overrides           | Configures default overrides for Kyma.                                                                                                      | Team Gopher     |
| create_runtime | BTPOperatorOverrides               | BTP                      | Configures the required credentials for BTP.                                                                                                | Team Gopher     |
| create_runtime | BusolaMigratorOverrides            | Busola                   | Sets configuration for Busola.                                                                                                              | Team Hasselhoff |
| create_runtime | Create_Runtime_Without_Kyma        | Provisioning             | Triggers provisioning of a Runtime in Runtime Provisioner.                                                                              | Team Gopher     |
| check_runtime  | Check_Runtime                      | Provisioning             | Checks the status of the Provisioner process and asks the Director for the Dashboard URL if the provisioning is completed in Gardener.      | Team Gopher     |
| create_runtime | Get_Kubeconfig                     | Provisioning             | Gets the kubeconfig.                                                                                                                        | Team Gopher     |
| create_runtime | Inject_BTP_Operator_Credentials    | Provisioning             | Creates a secret in the SKR with credentials for BTP.                                                                                         | Team Gopher     |
| create_runtime | Create_Cluster_Configuration       | Reconciler               | Applies the cluster configuration.                                                                                                          | Team Gopher     |
| check_kyma     | Check_Cluster_Configuration        | Reconciler               | Checks if the cluster configuration is applied .                                                                                            | Team Gopher     |
| create_kyma_resource | Apply_Kyma                         | Lifecycle Manager | Creates Kyma resource.                                                                                                                      | Team Gopher     |
| post_actions   | AVS_Create_External_Eval_Step      | AvS                      | Sets up external monitoring of Kyma Runtime.                                                                                                | Team Gopher     |
| post_actions   | AVS_Tags                           | AvS                      | Sets up proper tags in the internal monitoring system.                                                                                      | Team Gopher     |

The timeout for processing the whole provisioning operation is set to `24h`. In Kyma 2.0 provisioning steps delegate resource creation to Reconciler. Since Reconciler does not constrain a number of retries in case of a failed reconciliation, KEB sets [provisioning timeout for Reconciler](../../resources/kcp/charts/kyma-environment-broker/values.yaml#L49) to `2h`.

## Deprovisioning

Each deprovisioning step is responsible for a separate part of cleaning Runtime dependencies. To properly deprovision all Runtime dependencies, you need the data used during the Runtime provisioning. The first step finds the previous operation and copies the data.

None of the deprovisioning steps should block the entire deprovisioning operation. Use the `RetryOperationWithoutFail` function from the `DeprovisionOperationManager` struct to skip a step in case of a retry timeout. Set a 5-minute, at the most, timeout for retries in a step.
Each step has its own separate `stage`, so once the step is successfully executed, it won't be retried.
The deprovisioning process contains the following steps:

| Step                         | Domain                          | Description                                                                                                                                                  |
|------------------------------|---------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Init_Step                    | Deprovisioning                  | Changes the state from `pending` to `in progress` if there is no other operation in progress. It initializes the `InstanceDetails` from the last finished operation. |
| BTPOperator_Cleanup          | BTP                             | Deletes service instances and service bindings from the cluster.                                                                                             | 
| De-provision_AVS_Evaluations | AvS                             | Removes external and internal monitoring of Kyma Runtime.                                                                                                    |
| EDP_Deregistration           | Event Data Platform             | Removes all SKR entries from the Event Data Platform.                                                                                                        |
| IAS_Deregistration           | Identity Authentication Service | Removes the ServiceProvider from IAS.                                                                                                                        | 
| Deregister_Cluster           | Reconciler                      | Removes the cluster from Reconciler.                                                                                                                     | 
| Check_Cluster_Deregistration | Reconciler                      | Checks if the cluster deregistration is complete.                                                                                                            | 
| Remove_Runtime               | Provisioner                     | Triggers deprovisioning of a Runtime in Runtime Provisioner.                                                                                             | 
| Check_Runtime_Removal        | Provisioner                     | Checks if the cluster deprovisioning is complete.                                                                                                            |
| Release_Subscription         | Subscriptions                   | Releases the subscription used by the cluster.<br/>                                                                                                               |
| Remove_Instance              | Deprovisioning                  | Removes the instance from the database.                                                                                                                      |
>**NOTE:** The timeout for processing this operation is set to `24h`.

## Upgrade Kyma

Each upgrade step is responsible for a separate part of upgrading Runtime dependencies. To properly upgrade the Runtime, you need the data used during the Runtime provisioning. You can fetch this data from the **ProvisioningOperation** struct in the [initialization](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/internal/process/upgrade_kyma/initialisation.go) step.

The upgrade process contains the following steps:

| Step                                   | Description                                                                                           | Owner            |
|----------------------------------------|-------------------------------------------------------------------------------------------------------|------------------|
| Check_Cluster_Configuration            | Checks if the cluster configuration is applied                                                        | Team Gopher      |
| Get_Kubeconfig                         | Gets the kubeconfig file.                                                                             | Team Gopher      |
| BTPOperatorOverrides                   | Configures the required credentials for BTP.                                                          | Team Gopher      |
| Overrides_From_Secrets_And_Config_Step | Builds an input configuration that is passed as overrides to Runtime Provisioner.                     | Team Gopher      |
| BusolaMigratorOverrides                | Sets Busola's configuration.                                                                          | Team Hasselhoff  |
| Send_Notification                      | Notifies customers using SPC whenever an orchestration is scheduled, triggered, completed, or canceled. | Team SRE         |
| Apply_Cluster_Configuration            | Applies a cluster configuration to the Reconciler.                                                    | Team Gopher      |

>**NOTE:** The timeout for processing this operation is set to `3h`.

## Upgrade Cluster

| Step                          | Description                                                                                           |
|-------------------------------|-------------------------------------------------------------------------------------------------------|
| Send_Notification             | Notifies customers using SPC whenever an orchestration is scheduled, triggered, completed, or canceled. |
| Upgrade_Cluster               | Sends the updated cluster parameters to the Provisioner                                               |

## Update

| Stage               | Step                           | Description                                                                                   |
|---------------------|--------------------------------|-----------------------------------------------------------------------------------------------|
| cluster             | Update_Kyma_Initialization     | Changes the state from `pending` to `in progress` if there is no other operation in progress. |                                                                                                                     
| cluster             | Upgrade_Shoot                  | Sends the updated cluster parameters to the Provisioner.                                      |                                                                                                                     
| btp-operator        | Update_Init_Kyma_Version       | Specifies the Kyma version to install.                                                        |                                                                                                                      
| btp-operator        | Get_Kubeconfig                 | Gets the kubeconfig file.                                                                     |                                                                                                                      
| btp-operator        | BTPOperatorOverrides           | Configures the required credentials for BTP.                                                  |                                                                                                                     
| btp-operator        | Apply_Reconciler_Configuration | Applies the cluster configuration to the Reconciler.                                          |                                                                                                                      
| btp-operator-check  | CheckReconcilerState           | Checks if the cluster configuration is applied                                                |                                                                                                                      
| check               | Check_Runtime                  | Checks the status of the Provisioner process.                                                 |                                                                                                                      


## Provide additional steps

You can configure Runtime operations by providing additional steps. To add a new step, follow these tutorials:

<div tabs name="runtime-provisioning-deprovisioning" group="runtime-provisioning-deprovisioning">
  <details>
  <summary label="provisioning">
  Provisioning
  </summary>

1. Create a new file in [this directory](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/internal/process/provisioning).

2. Implement the following interface in your provisioning or deprovisioning step:

    ```go
    type Step interface {
        Name() string
        Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error)
    }
    ```

   - `Name()` method returns the name of the step that is used in logs.
   - `Run()` method implements the functionality of the step. The method receives operations as an argument to which it can add appropriate overrides or save other used variables.


    ```go
    operation.InputCreator.SetOverrides(COMPONENT_NAME, []*gqlschema.ConfigEntryInput{
        {
            Key:   "path.to.key",
            Value: SOME_VALUE,
        },
        {
            Key:    "path.to.secret",
            Value:  SOME_VALUE,
            Secret: ptr.Bool(true),
        },
    })
    ```

    If your functionality contains long-term processes, you can store data in the storage.
    To do this, add the following field to the provisioning operation in which you want to save data:

    ```go
    type Operation struct {

        // These fields are serialized to JSON and stored in the storage
        RuntimeVersion RuntimeVersionData `json:"runtime_version"`

        // These fields are not stored in the storage
        InputCreator ProvisionerInputCreator `json:"-"`
    }
    ```

    By saving data in the storage, you can check if you already have the necessary data and avoid time-consuming processes. You must always return the modified operation from the method.

    See the example of the step implementation:

    ```go
    package provisioning

    import (
        "encoding/json"
        "net/http"
        "time"

        "github.com/kyma-incubator/compass/components/provisioner/pkg/gqlschema"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal/storage"

        "github.com/sirupsen/logrus"
    )

    type HelloWorldStep struct {
        operationStorage storage.Operations
        client           *http.Client
    }

    type ExternalBodyResponse struct {
        data  string
        token string
    }

    func NewHelloWorldStep(operationStorage storage.Operations, client *http.Client) *HelloWorldStep {
        return &HelloWorldStep{
            operationStorage: operationStorage,
            client:           client,
        }
    }

    func (s *HelloWorldStep) Name() string {
        return "Hello_World"
    }

    // Your step can be repeated in case any other step fails, even if your step has already done its job
    func (s *HelloWorldStep) Run(operation internal.Operation, log *logrus.Entry) (internal.Operation, time.Duration, error) {
        log.Info("Start step")

        // Check whether your step should be run or if its job has been done in the previous iteration
        // All non-save operation data are empty (e.g. InputCreator overrides)

        // Add your logic here

        // Add a call to an external service (optional)
        response, err := s.client.Get("http://example.com")
        if err != nil {
            // Error during a call to an external service may be temporary so you should return time.Duration
            // All steps will be repeated in X seconds/minutes
            return operation, 1 * time.Second, nil
        }
        defer response.Body.Close()

        body := ExternalBodyResponse{}
        err = json.NewDecoder(response.Body).Decode(&body)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If a call or any other action is time-consuming, you can save the result in the operation
        // If you need an extra field in the Operation structure, add it first
        // In the following step, you can check beforehand if a given value already exists in the operation
        operation.HelloWorlds = body.data
        updatedOperation, err := s.operationStorage.UpdateOperation(operation)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If your step finishes with data which should be added to override used during the Runtime provisioning,
        // add an extra value to operation.InputCreator, then return the updated version of the Application
        updatedOperation.InputCreator.SetOverrides("component-name", []*gqlschema.ConfigEntryInput{
            {
                Key:   "some.key",
                Value: body.token,
            },
        })

        // Return the updated version of the Application
        return *updatedOperation, 0, nil
    }
    ```

3. Add the step to the [`/cmd/broker/main.go`](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/cmd/broker/main.go) file:

    ```go
    provisioningSteps := []struct {
   		stage   string
   		step     provisioning.Step
   	}{
   		{
   			stage: "create_runtime",
   			step:   provisioning.NewHelloWorldStep(db.Operations(), &http.Client{}),
   		},
    }
    ```

   Once all the steps in the stage have run successfully, the stage is  not retried even if the application is restarted.

  </details>

  <details>
  <summary label="upgrade">
  Upgrade
  </summary>

1. Create a new file in [this directory](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/internal/process/kyma_upgrade).

2. Implement the following interface in your upgrade step:

    ```go
    type Step interface {
        Name() string
        Run(operation internal.UpgradeOperation, logger logrus.FieldLogger) (internal.UpgradeOperation, time.Duration, error)
    }
    ```

   - `Name()` method returns the name of the step that is used in logs.
   - `Run()` method implements the functionality of the step. The method receives operations as an argument to which it can add appropriate overrides or save other used variables.


    If your functionality contains long-term processes, you can store data in the storage.
    To do this, add this field to the upgrade operation in which you want to save data:

    ```go
    type UpgradeOperation struct {
        Operation `json:"-"`

        // add additional data here
    }
    ```

    By saving data in the storage, you can check if you already have the necessary data and avoid time-consuming processes. You should always return the modified operation from the method.

    See the example of the step implementation:

    ```go
    package upgrade

    import (
        "encoding/json"
        "net/http"
        "time"

        "github.com/kyma-incubator/compass/components/provisioner/pkg/gqlschema"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal/storage"

        "github.com/sirupsen/logrus"
    )

    type HelloWorldStep struct {
        operationStorage storage.Operations
        client           *http.Client
    }

    type ExternalBodyResponse struct {
        data  string
        token string
    }

    func NewHelloWorldStep(operationStorage storage.Operations, client *http.Client) *HelloWorldStep {
        return &HelloWorldStep{
            operationStorage: operationStorage,
            client:           client,
        }
    }

    func (s *HelloWorldStep) Name() string {
        return "Hello_World"
    }

    // Your step can be repeated in case any other step fails, even if your step has already done its job
    func (s *HelloWorldStep) Run(operation internal.UpgradeOperation, log *logrus.Entry) (internal.UpgradeOperation, time.Duration, error) {
        log.Info("Start step")

        // Check whether your step should be run or if its job has been done in the previous iteration
        // All non-save operation data are empty (e.g. InputCreator overrides)

        // Add your logic here

        // Add a call to an external service (optional)
        response, err := s.client.Get("http://example.com")
        if err != nil {
            // Error during a call to an external service may be temporary so you should return time.Duration
            // All steps will be repeated in X seconds/minutes
            return operation, 1 * time.Second, nil
        }
        defer response.Body.Close()

        body := ExternalBodyResponse{}
        err = json.NewDecoder(response.Body).Decode(&body)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If a call or any other action is time-consuming, you can save the result in the operation
        // If you need an extra field in the UpgradeOperation structure, add it first
        // in the step below; beforehand, you can check if a given value already exists in the operation
        operation.HelloWorlds = body.data
        updatedOperation, err := s.operationStorage.UpdateUpgradeOperation(operation)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If your step finishes with data which should be added to override used during the Runtime upgrade,
        // add an extra value to operation.InputCreator, then return the updated version of the Application
        updatedOperation.InputCreator.SetOverrides("component-name", []*gqlschema.ConfigEntryInput{
            {
                Key:   "some.key",
                Value: body.token,
            },
        })

        // Return the updated version of the Application
        return *updatedOperation, 0, nil
    }
    ```

3. Add the step to the [`/cmd/broker/main.go`](https://github.com/kyma-project/control-plane/blob/main/components/kyma-environment-broker/cmd/broker/main.go) file:

    ```go
    upgradeSteps := []struct {
   		weight   int
   		step     upgrade_kyma.Step
   	}{
   		{
   			weight: 1,
   			step:   upgrade_kyma.NewHelloWorldStep(db.Operations(), &http.Client{}),
   		},
    }
    ```

   </details>
</div>

## Stages

An operation defines stages and steps which represent the work you must do. A stage is a grouping unit for steps. A step is a part of a stage. An operation can consist of multiple stages, and a stage can consist of multiple steps. You group steps in a stage when you have some sensitive data which you don't want to store in database. In such a case you temporarily store the sensitive data in the memory and go through the steps. Once all the steps in a stage are successfully executed, the stage is marked as finished and never repeated again, even if the next one fails. If any steps fail at a given stage, the whole stage is repeated from the beginning.