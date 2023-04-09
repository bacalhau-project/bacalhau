package s3

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/rs/zerolog/log"
)

var isEC2Instance bool
var isEC2InstanceOnce sync.Once
var isEC2InstanceTimeout = 2 * time.Second

func DefaultAWSConfig() (aws.Config, error) {
	// Set a default IMDC TTL of 1 hour if not set to avoid hitting the metadata service too often, which can slow down the
	// node's startup time.
	if _, ok := os.LookupEnv("AWS_EC2_METADATA_TTL"); !ok {
		err := os.Setenv("AWS_EC2_METADATA_TTL", "3600")
		if err != nil {
			return aws.Config{}, err
		}
	}
	var optFns []func(*config.LoadOptions) error
	return config.LoadDefaultConfig(context.Background(), optFns...)
}

// HasValidCredentials returns true if the AWS config has valid credentials.
func HasValidCredentials(config aws.Config) bool {
	credentials, err := config.Credentials.Retrieve(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to check if we have valid AWS credentials")
		return false
	}
	return credentials.HasKeys()
}

// IsEC2Instance returns true if the current process is running on an EC2 instance.
// This requires checking the EC2 instance metadata service, which takes a few seconds to resolve. This is why we are only calling it
// once and caching the result.
func IsEC2Instance(ctx context.Context) bool {
	isEC2InstanceOnce.Do(func() {
		ctx2, cancel := context.WithTimeout(ctx, isEC2InstanceTimeout)
		defer cancel()
		cfg, err := config.LoadDefaultConfig(ctx2)
		if err != nil {
			isEC2Instance = false
			return
		}

		client := imds.NewFromConfig(cfg)
		_, err = client.GetMetadata(ctx2, &imds.GetMetadataInput{
			Path: "instance-id",
		})
		if err != nil {
			isEC2Instance = false
		}
		isEC2Instance = true
	})
	return isEC2Instance
}
