package cloudprovider

type gcpResourceCleaner struct {
}

func NewGCPeResourcesCleaner(secretData map[string][]byte) ResourceCleaner {
	return &azureResourceCleaner{}
}

func (rc gcpResourceCleaner) Do() error {
	return nil
}
