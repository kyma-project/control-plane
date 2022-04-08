package cloudprovider

type awsResourceCleaner struct {
}

type awsConfig struct {
	accessKeyID     string
	secretAccessKey string
}

func NewAwsResourcesCleaner(secretData map[string][]byte) (ResourceCleaner, error) {
	return awsResourceCleaner{}, nil
}

func (ac awsResourceCleaner) Do() error {
	// TODO: clean up resources if required...
	return nil
}
