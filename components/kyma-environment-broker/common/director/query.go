package director

import "fmt"

const (
	instanceIDLabelKey = "broker_instance_id"
)

type queryProvider struct{}

func (qp queryProvider) Runtime(runtimeID string) string {
	return fmt.Sprintf(`query {
	result: runtime(id: "%s") {
	%s
	}
}`, runtimeID, runtimeStatusData())
}

func (qp queryProvider) SetRuntimeLabel(runtimeId, key, value string) string {
	return fmt.Sprintf(`mutation {
		result: setRuntimeLabel(runtimeID: "%s", key: "%s", value: "%s") {
			%s
		}
	}`, runtimeId, key, value, labelData())
}

func (qp queryProvider) RuntimeForInstanceId(instanceID string) string {
	return fmt.Sprintf(`query {
	result: runtimes(filter: { key: "%s" query: "\"%s\"" }) {
    data {
      id
	}
}
}`, instanceIDLabelKey, instanceID)
}

func runtimeStatusData() string {
	return `id
			status{
				condition
			}`
}

func labelData() string {
	return `key
			value`
}
