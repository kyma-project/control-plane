package cloudprovider

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const defaultRegion string = "eu-central-1"

type awsResourceCleaner struct {
	ec2Client   ec2.Client
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
	awsEC2Client, err := awsResourceCleaner.newAwsEC2Client(awsConfig, defaultRegion)
	if err != nil {
		return nil, err
	}

	awsResourceCleaner.ec2Client = awsEC2Client
	return awsResourceCleaner, nil
}

func (ac awsResourceCleaner) Do() error {
	all_regions := ac.getAllRegions()

	// Iterate through all regions
	for _, region := range all_regions.Regions {
		log.Printf("Switching to region %v", *region.RegionName)
		newEc2, err := ac.newAwsEC2Client(ac.credentials, *region.RegionName)
		ac.ec2Client = newEc2

		// Iterate through volumes
		volumes, err := ac.ec2Client.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{})
		if err != nil {
			return err
		}

		log.Println("Listing Volumes:")
		for _, volume := range volumes.Volumes {
			log.Printf("Volume ID: %v", *volume.VolumeId)
		}

		ac.deleteVolumes()
	}

	return nil
}

func (ac awsResourceCleaner) deleteVolumes() error {
	volumes, err := ac.ec2Client.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		log.Printf("Deleting volume with id: %v", *volume.VolumeId)
		if volume.State == types.VolumeStateInUse {
			log.Printf("Volume is in-use, detaching it first: ")
			ac.ec2Client.DetachVolume(context.TODO(), &ec2.DetachVolumeInput{
				VolumeId: volume.VolumeId,
			})
		}

		log.Printf("Deleting volume %v", *volume.VolumeId)
		ac.ec2Client.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{
			VolumeId: volume.VolumeId,
		})

	}

	return nil
}

func (ac awsResourceCleaner) getAllRegions() ec2.DescribeRegionsOutput {
	allRegions := false
	regionOutput, err := ac.ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{AllRegions: &allRegions})
	if err != nil {
		return ec2.DescribeRegionsOutput{}
	}

	return *regionOutput
}

func (ac awsResourceCleaner) toAwsConfig(secretData map[string][]byte) (awsCredentialsConfig, error) {
	accessKeyID, exists := secretData["accessKeyID"]
	if !exists {
		return awsCredentialsConfig{}, errors.New("AccessKeyID was not provided in secret!")
	}

	secretAccessKey, exists := secretData["secretAccessKey"]
	if !exists {
		return awsCredentialsConfig{}, errors.New("SecretAccessKey was not provided in secret!")
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
