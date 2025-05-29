package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type AMIBuilder struct {
	client        *ec2.Client
	instanceID    string
	region        string
	githubToken   string
	chuckdRepoURL string
	baseAMI       string
}

func NewAMIBuilder(ctx context.Context, region, githubToken, chuckdRepoURL, baseAMI string) (*AMIBuilder, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading aws sdk config: %v", err)
	}

	return &AMIBuilder{
		client:        ec2.NewFromConfig(cfg),
		region:        region,
		githubToken:   githubToken,
		chuckdRepoURL: chuckdRepoURL,
		baseAMI:       baseAMI,
	}, nil
}

func (b *AMIBuilder) getBuildScript() string {
	// User data script to install Go, clone repo, and build binary
	script := fmt.Sprintf(`#!/bin/bash
set -e

# Create log file for this script
LOG_FILE="/var/log/groundhog-build-script.log"
exec > >(tee -a $LOG_FILE)
exec 2>&1

# Update system
apt-get update
apt-get upgrade -y

# Install dependencies
apt-get install -y wget git build-essential

# Setup git for private repos
git config --global url."https://%s:@github.com/".insteadOf "https://github.com/"
git config --global url."https://%s@github.com/".insteadOf "https://github.com/"

# Install latest Go
GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -n1 | xargs)
wget https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf ${GO_VERSION}.linux-amd64.tar.gz
rm ${GO_VERSION}.linux-amd64.tar.gz

# Setup go for private repos
go env -w GOPRIVATE=github.com/groundhogchain/*

# Set up Go environment
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
echo 'export GOPATH=/root/go' >> /etc/profile
echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile
source /etc/profile

# Clone private repository
git clone https://%s@%s /tmp/chuckd

# Build and install chuckd
cd /tmp/chuckd/chuckd
/usr/local/go/bin/go build -o /usr/local/bin/chuckd .
chmod +x /usr/local/bin/chuckd

# Clean up
rm -rf /tmp/chuckd

# Signal completion
echo "Setup completed successfully" > /tmp/setup-complete.txt
`,
		b.githubToken, b.githubToken,
		b.githubToken, b.chuckdRepoURL)

	return base64.StdEncoding.EncodeToString([]byte(script))
}

func (b *AMIBuilder) createSecurityGroup(ctx context.Context) (string, error) {
	sgName := fmt.Sprintf("ami-builder-sg-%d", time.Now().Unix())

	createSGInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String("Temporary security group for AMI building"),
	}

	result, err := b.client.CreateSecurityGroup(ctx, createSGInput)
	if err != nil {
		return "", fmt.Errorf("creating security group: %v", err)
	}

	// Add SSH access (you might want to restrict this to your IP)
	authorizeInput := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: result.GroupId,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"), // Consider restricting this
						Description: aws.String("SSH access"),
					},
				},
			},
		},
	}

	_, err = b.client.AuthorizeSecurityGroupIngress(ctx, authorizeInput)
	if err != nil {
		return "", fmt.Errorf("authorizing security group ingress: %v", err)
	}

	return *result.GroupId, nil
}

func (b *AMIBuilder) launchInstance(ctx context.Context, securityGroupID string) error {

	encodedBuildScript := base64.StdEncoding.EncodeToString([]byte(b.getBuildScript()))

	runInput := &ec2.RunInstancesInput{
		ImageId:          aws.String(b.baseAMI),
		InstanceType:     types.InstanceTypeT2Micro, // Change as needed
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		UserData:         aws.String(encodedBuildScript),
		SecurityGroupIds: []string{securityGroupID},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(fmt.Sprintf("Groundhog AMI Builder %s", time.Now().Format(time.RFC1123))),
					},
				},
			},
		},
	}

	result, err := b.client.RunInstances(ctx, runInput)
	if err != nil {
		return fmt.Errorf("launching instance: %v", err)
	}

	b.instanceID = *result.Instances[0].InstanceId
	liblog.Infof("launched instance: %s", b.instanceID)

	return nil
}

func (b *AMIBuilder) getInstancePublicIP(ctx context.Context) (*string, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{b.instanceID},
	}

	result, err := b.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describing instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	return instance.PublicIpAddress, nil
}

func (b *AMIBuilder) waitForInstance(ctx context.Context) error {
	waiter := ec2.NewInstanceStatusOkWaiter(b.client)

	liblog.Infof("Waiting for instance to be ready...")
	err := waiter.Wait(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{b.instanceID},
	}, 10*time.Minute)

	if err != nil {
		return fmt.Errorf("waiting for instance: %v", err)
	}

	return nil
}

func (b *AMIBuilder) createAMI(ctx context.Context, amiName string) (string, error) {
	createImageInput := &ec2.CreateImageInput{
		InstanceId:  aws.String(b.instanceID),
		Name:        aws.String(amiName),
		Description: aws.String("Groundhog AMI"),
		NoReboot:    aws.Bool(false), // Reboot for consistency
	}

	result, err := b.client.CreateImage(ctx, createImageInput)
	if err != nil {
		return "", fmt.Errorf("creating AMI: %v", err)
	}

	liblog.Infof("Creating AMI: %s", *result.ImageId)

	// Wait for AMI to be available
	waiter := ec2.NewImageAvailableWaiter(b.client)
	err = waiter.Wait(
		ctx,
		&ec2.DescribeImagesInput{
			ImageIds: []string{*result.ImageId},
		},
		15*time.Minute,
	)

	if err != nil {
		return "", fmt.Errorf("waiting for AMI: %v", err)
	}

	return *result.ImageId, nil
}

func (b *AMIBuilder) cleanup(ctx context.Context, securityGroupID string) {

	liblog.Infof("Cleaning up...")

	// Terminate instance
	if b.instanceID != "" {
		_, err := b.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{b.instanceID},
		})
		if err != nil {
			liblog.Errorf("terminating instance: %v", err)
		} else {
			liblog.Infof("terminated instance: %s", b.instanceID)
		}
	}

	// Wait a bit for instance to terminate before deleting security group
	time.Sleep(30 * time.Second)

	// Delete security group
	if securityGroupID != "" {
		_, err := b.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(securityGroupID),
		})
		if err != nil {
			liblog.Errorf("deleting security group: %v", err)
		} else {
			liblog.Infof("deleted security group: %s", securityGroupID)
		}
	}
}

func (b *AMIBuilder) Build(ctx context.Context, amiName string) (string, error) {

	sgID, err := b.createSecurityGroup(ctx)
	if err != nil {
		return "", err
	}
	defer b.cleanup(ctx, sgID)

	err = b.launchInstance(ctx, sgID)
	if err != nil {
		return "", err
	}

	err = b.waitForInstance(ctx)
	if err != nil {
		return "", err
	}

	// Additional wait for user data script to complete
	liblog.Infof("Waiting for setup script to complete...")
	time.Sleep(5 * time.Minute) // XXX: Adjust based on your setup complexity

	pip, err := b.getInstancePublicIP(ctx)
	if err != nil {
		return "", err
	}
	if pip == nil {
		return "", fmt.Errorf("ec2 instance has no public IP")
	}
	liblog.Infof("log into builder: ssh -i YOUR.pem ubuntu@%s", *pip)

	amiID, err := b.createAMI(ctx, amiName)
	if err != nil {
		return "", err
	}

	return amiID, nil
}

func BuildGroundhogAMI(ctx context.Context, region, githubToken, baseAMI string) {

	// Configuration
	chuckdRepoURL := "github.com/groundhogchain/chuckd.git" // Your private repo (without https://)
	amiName := fmt.Sprintf("groundhog-%d", time.Now().Unix())

	// Create builder
	builder, err := NewAMIBuilder(ctx, region, githubToken, chuckdRepoURL, baseAMI)
	if err != nil {
		liblog.Fatalf("creating builder: %v", err)
	}

	// Build AMI
	liblog.Infof("Starting AMI build process...")
	amiID, err := builder.Build(ctx, amiName)
	if err != nil {
		liblog.Fatalf("building: %v", err)
	}

	liblog.Infof("Successfully created AMI: %s", amiID)
}

func FindBaseAMI(ctx context.Context, region string) (string, error) {

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("loading aws sdk config: %v", err)
	}
	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeImagesInput{
		Owners: []string{"099720109477"}, // Canonical's AWS account ID
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	}

	result, err := client.DescribeImages(ctx, input)
	if err != nil {
		return "", err
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no Ubuntu AMIs found")
	}

	// Find the most recent
	var latestAMI types.Image
	for _, image := range result.Images {
		if latestAMI.CreationDate == nil || *image.CreationDate > *latestAMI.CreationDate {
			latestAMI = image
		}
	}

	return *latestAMI.ImageId, nil
}
