How to begin?

```bash
> go mod vendor
> env GO111MODULE=off go run *.go 
```

There are five test steps, each works for some time (sleep for X seconds)
Two of them (step: 2,4) failed during execution and are repeated. Step 3 will pass
only if steps before did their job.

Each step has the same timeout, set in Manager. To watch how timeout works decrease timeout 
in Manager (stepTimeout field).

Description of the elements:
- `/internal/model.go` - short version of the operation/prrovisioning operation from KEB
- `/internal/storage` - short version of memory storage from KEB
- `/internal/process` - **new queue and provisioning manager**
- `/test` - test steps plus queue execution
```bash
> env GO111MODULE=off go test ./...
```