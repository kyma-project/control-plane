package model

import "time"

const (
	AWS = "aws"
)

const (
	ProductionProfile KymaProfile = "PRODUCTION"
)

type KymaProfile string

type AWSProviderConfigInput struct {
	VpcCidr  string          `json:"vpcCidr"`
	AwsZones []*AWSZoneInput `json:"zones"`
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

type Cluster struct {
	ID                string
	CreationTimestamp time.Time
	Tenant            string
	SubAccountId      *string
	KymaConfigID      string
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

type KymaConfig struct {
	ID        string
	Release   Release
	Profile   KymaProfile
	ClusterID string
	Active    bool
}

type Release struct {
	Id            string
	Version       string
	TillerYAML    string
	InstallerYAML string
}
