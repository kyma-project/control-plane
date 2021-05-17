package model

const (
	AWS = "aws"
)

type AWSProviderConfigInput struct {
	VpcCidr string          `json:"vpcCidr"`
	Zones   []*AWSZoneInput `json:"zones"`
}

type AWSZoneInput struct {
	Name         string `json:"name"`
	PublicCidr   string `json:"publicCidr"`
	InternalCidr string `json:"internalCidr"`
	WorkerCidr   string `json:"workerCidr"`
}

type OldAWSProviderConfigInput struct {
	Zone         string `json:"zone"`
	VpcCidr      string `json:"vpcCidr"`
	PublicCidr   string `json:"publicCidr"`
	InternalCidr string `json:"internalCidr"`
}
