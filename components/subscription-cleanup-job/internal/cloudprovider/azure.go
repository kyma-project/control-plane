package cloudprovider

type azureResourceCleaner struct {
}

func NewAzureResourcesCleaner(secretData map[string][]byte) ResourceCleaner {
	return &azureResourceCleaner{}
}

func (rc azureResourceCleaner) Do() error {
	return nil
}
