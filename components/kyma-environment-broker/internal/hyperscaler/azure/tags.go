package azure

const (
	// Azure tags used to identify the Kyma runtime
	TagSubAccountID = "SubAccountID"
	TagInstanceID   = "InstanceID"
	TagOperationID  = "OrchestrationID"
)

// Tags type represents Azure tags acceptable by the Azure client
type Tags map[string]*string
