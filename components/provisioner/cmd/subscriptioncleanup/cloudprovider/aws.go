package cloudprovider

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type awsResourceCleaner struct {
	credentials awsCredentialsConfig
}

type awsCredentialsConfig struct {
	accessKeyID     string
	secretAccessKey string
}

func NewAwsResourcesCleaner(secretData map[string][]byte) (ResourceCleaner, error) {
	awsResourceCleaner := awsResourceCleaner{}
	awsConfig, err := awsResourceCleaner.toAwsConfig(secretData)
	if err != nil {
		return nil, err
	}

	awsResourceCleaner.credentials = awsConfig
	return awsResourceCleaner, nil
}

func (ac awsResourceCleaner) Do() error {
	all_regions, err := ac.getAllRegions()
	if err != nil {
		return err
	}

	for _, region := range all_regions.Regions {
		logrus.Printf("Switching to region %v", *region.RegionName)
		ec2Client, err := ac.newAwsEC2Client(ac.credentials, *region.RegionName)
		if err != nil {
			return err
		}

		err = ac.deleteVolumes(ec2Client)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ac awsResourceCleaner) deleteVolumes(ec2Client ec2.Client) error {
	volumes, err := ec2Client.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if volume.State == types.VolumeStateInUse {
			return fmt.Errorf(fmt.Sprintf("There is an EC2 instance which uses this volume with id: %v", *volume.VolumeId))
		}
	}

	for _, volume := range volumes.Volumes {
		logrus.Printf("Deleting volume with id %v", *volume.VolumeId)
		ec2Client.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{
			VolumeId: volume.VolumeId,
		})
	}

	return nil
}

func (ac awsResourceCleaner) getAllRegions() (ec2.DescribeRegionsOutput, error) {
	allRegions := false
	ec2Client, err := ac.newAwsEC2Client(ac.credentials, "eu-central-1")
	if err != nil {
		return ec2.DescribeRegionsOutput{}, err
	}

	regionOutput, err := ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{AllRegions: &allRegions})
	if err != nil {
		return ec2.DescribeRegionsOutput{}, err
	}

	return *regionOutput, nil
}

func (ac awsResourceCleaner) toAwsConfig(secretData map[string][]byte) (awsCredentialsConfig, error) {
	accessKeyID, exists := secretData["accessKeyID"]
	if !exists {
		return awsCredentialsConfig{}, fmt.Errorf("AccessKeyID was not provided in secret!")
	}

	secretAccessKey, exists := secretData["secretAccessKey"]
	if !exists {
		return awsCredentialsConfig{}, fmt.Errorf("SecretAccessKey was not provided in secret!")
	}

	return awsCredentialsConfig{
		accessKeyID:     string(accessKeyID),
		secretAccessKey: string(secretAccessKey),
	}, nil
}

func (ac awsResourceCleaner) newAwsEC2Client(awsCredentialConfig awsCredentialsConfig, region string) (ec2.Client, error) {
	return *ec2.New(ec2.Options{
		Region:      region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(awsCredentialConfig.accessKeyID, awsCredentialConfig.secretAccessKey, "")),
	}), nil
}
