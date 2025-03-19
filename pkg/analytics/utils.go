package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// GetDockerImageTelemetry returns the Docker image name for telemetry purposes.
// For trusted images (from Bacalhau or Expanso), it returns the original image name.
// For non-trusted images, it returns a hashed version of the image name for privacy.
// Returns an empty string if the engine is not Docker or no image information is found.
func GetDockerImageTelemetry(engineParams *models.SpecConfig) string {
	// Only process Docker engines
	if engineParams == nil || engineParams.Type != models.EngineDocker {
		return ""
	}

	// Try both "Image" and "image" keys
	imageParam, ok := engineParams.Params["Image"]
	if !ok {
		imageParam, ok = engineParams.Params["image"]
		if !ok {
			return ""
		}
	}

	// Convert to string (should always be a string but handle just in case)
	imageName, ok := imageParam.(string)
	if !ok || imageName == "" {
		return ""
	}

	// Check if image is from a trusted source (Bacalhau or Expanso)
	trusted := false
	trustedPrefixes := []string{
		"ghcr.io/bacalhau-project/",
		"expanso/",
		"bacalhauproject/",
	}

	for _, prefix := range trustedPrefixes {
		if strings.HasPrefix(strings.ToLower(imageName), strings.ToLower(prefix)) {
			trusted = true
			break
		}
	}

	if trusted {
		return imageName
	} else {
		// For non-trusted images, hash the image name for privacy
		return hashString(imageName)
	}
}

// hashString creates a SHA-256 hash of the input string and returns its hexadecimal representation.
// Used for privacy-preserving analytics.
func hashString(in string) string {
	hash := sha256.New()
	hash.Write([]byte(in))
	return hex.EncodeToString(hash.Sum(nil))
}
