package s3managed

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// ValidatePublisherConfig checks if the provided S3 managed publisher configuration is valid.
// It ensures that the mandatory paramenters (bucket, region, and pre-signed URL expiration) are set correctly.
func ValidatePublisherConfig(p types.S3ManagedPublisher) error {
	var errs []string

	if p.Bucket == "" {
		errs = append(errs, "bucket cannot be empty")
	}

	if p.Region == "" {
		errs = append(errs, "region cannot be empty")
	}

	if p.PreSignedURLExpiration.AsTimeDuration() <= 0 {
		errs = append(errs, "pre-signed URL expiration must be greater than zero")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid publisher configuration: %s", strings.Join(errs, ", "))
	}

	return nil
}
