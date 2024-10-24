package s3

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/rs/zerolog/log"
)

// IMDS (Instance Metadata Service) availability check results are cached globally
// to avoid repeated timeout delays when IMDS is not available, which is the case
// when running outside of AWS or when IMDS is disabled.
var (
	// imdsAvailable indicates whether IMDS is accessible
	// This is set once during the first check and never modified afterwards
	imdsAvailable bool

	// imdsCheckOnce ensures the IMDS check is performed exactly once
	imdsCheckOnce sync.Once
)

// checkIMDSAvailability determines if the AWS Instance Metadata Service (IMDS) is
// accessible from the current environment. This function is safe to call multiple
// times - the actual check will only be performed once, with subsequent calls
// returning the cached result.
//
// Returns:
//   - true if IMDS is available and responding
//   - false if IMDS is disabled, unavailable, or times out
func checkIMDSAvailability() bool {
	imdsCheckOnce.Do(func() {
		// Check if IMDS is explicitly disabled
		// such as if `AWS_EC2_METADATA_DISABLED` is set to "true"
		imdsOptions := imds.Options{}
		if imdsOptions.ClientEnableState == imds.ClientDisabled {
			imdsAvailable = false
			return
		}

		// Attempt to access IMDS with configured timeout or default to 1 second
		timeout := 1 * time.Second
		if timeoutStr := os.Getenv("AWS_METADATA_SERVICE_TIMEOUT"); timeoutStr != "" {
			if timeoutInt, err := strconv.Atoi(timeoutStr); err == nil && timeoutInt > 0 {
				timeout = time.Duration(timeoutInt) * time.Second
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client := imds.New(imdsOptions)
		_, err := client.GetMetadata(ctx, &imds.GetMetadataInput{
			Path: "instance-id",
		})

		imdsAvailable = err == nil
		if !imdsAvailable {
			log.Debug().Msg("IMDS not available, will skip in future credential checks")
		}
	})

	return imdsAvailable
}

// DefaultAWSConfig returns an AWS configuration with IMDS disabled if it's
// determined to be unavailable or inaccessible. This prevents delays from
// failed IMDS calls during credential retrieval.
func DefaultAWSConfig() (aws.Config, error) {
	var optFns []func(*config.LoadOptions) error

	// If IMDS is not available, disable it in the config
	if !checkIMDSAvailability() {
		optFns = append(optFns, config.WithEC2IMDSClientEnableState(imds.ClientDisabled))
	}

	return config.LoadDefaultConfig(context.Background(), optFns...)
}

// HasValidCredentials returns true if the AWS config has valid credentials.
func HasValidCredentials(config aws.Config) bool {
	credentials, err := config.Credentials.Retrieve(context.Background())
	if err != nil {
		// Only log if it's not an expected IMDS disabled error
		if !strings.Contains(err.Error(), "EC2 IMDS") {
			log.Debug().Err(err).Msg("Failed to check if we have valid AWS credentials")
		}
		return false
	}
	return credentials.HasKeys()
}

func CanRunS3Test() bool {
	cfg, err := DefaultAWSConfig()
	if err != nil {
		return false
	}

	return HasValidCredentials(cfg)
}

// IsAWSEndpoint checks if the given S3 endpoint URL is an AWS endpoint by its suffix.
// If the endpoint is empty, it is considered an AWS endpoint.
func IsAWSEndpoint(endpoint string) bool {
	return endpoint == "" || strings.HasSuffix(endpoint, "amazonaws.com")
}
