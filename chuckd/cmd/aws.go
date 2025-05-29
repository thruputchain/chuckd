package cmd

import (
	"fmt"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/deploy/aws"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/spf13/cobra"
)

var (
	awsCmd = &cobra.Command{
		Use:   "aws",
		Short: "AWS provisioning and deployment tools",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	awsBuildAMICmd = &cobra.Command{
		Use:   "build-ami",
		Short: "build groundhog AMI",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				aws.BuildGroundhogAMI(ctx, awsRegion, awsGithubToken, awsBaseAMI)
			})
		},
	}

	awsFindBaseAMICmd = &cobra.Command{
		Use:   "find-base-ami",
		Short: "find base AMI",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				ami, err := aws.FindBaseAMI(ctx, awsRegion)
				if err != nil {
					liblog.Fatalf("finding ami: %v", err)
				}
				fmt.Println(ami)
			})
		},
	}
)

var (
	awsRegion      string
	awsGithubToken string
	awsBaseAMI     string
)

func init() {
	rootCmd.AddCommand(awsCmd)

	awsCmd.AddCommand(awsBuildAMICmd)
	awsBuildAMICmd.Flags().StringVar(&awsRegion, "region", "us-west-1", "AWS region")
	awsBuildAMICmd.Flags().StringVar(&awsGithubToken, "github-token", "", "GitHub token")
	awsBuildAMICmd.Flags().StringVar(&awsBaseAMI, "base-ami", "", "base AMI")

	awsCmd.AddCommand(awsFindBaseAMICmd)
	awsFindBaseAMICmd.Flags().StringVar(&awsRegion, "region", "us-west-1", "AWS region")
}
