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

type GardenerConfig struct {
	ID                                  string
	ClusterID                           string
	Name                                string
	ProjectName                         string
	KubernetesVersion                   string
	VolumeSizeGB                        *int
	DiskType                            *string
	MachineType                         string
	MachineImage                        *string
	MachineImageVersion                 *string
	Provider                            string
	Purpose                             *string
	LicenceType                         *string
	Seed                                string
	TargetSecret                        string
	Region                              string
	WorkerCidr                          string
	AutoScalerMin                       int
	AutoScalerMax                       int
	MaxSurge                            int
	MaxUnavailable                      int
	EnableKubernetesVersionAutoUpdate   bool
	EnableMachineImageVersionAutoUpdate bool
	AllowPrivilegedContainers           bool
	GardenerProviderConfig              string
}
