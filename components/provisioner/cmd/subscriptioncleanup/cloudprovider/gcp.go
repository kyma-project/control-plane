package cloudprovider

type gcpResourceCleaner struct {
}

func NewGCPeResourcesCleaner(secretData map[string][]byte) ResourceCleaner {
	return &gcpResourceCleaner{}
}

func (rc gcpResourceCleaner) Do() error {
	return nil
}
