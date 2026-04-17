package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// getDockerImageAnalytics returns the Docker image name for analytics purposes.
// For trusted images (from Bacalhau or Expanso), it returns the original image name.
// For non-trusted images, it returns a hashed version of the image name for privacy.
// Returns an empty string if the engine is not Docker or no image information is found.
func getDockerImageAnalytics(engineParams *models.SpecConfig) string {
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

// hashString returns a SHA256 hash of the input string.
func hashString(s string) string {
	if s == "" {
		return ""
	}
	hash := sha256.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

// getInputSourceTypes extracts the source types from a task's input sources.
// Returns an empty slice if the task has no input sources.
func getInputSourceTypes(t *models.Task) []string {
	if t == nil || len(t.InputSources) == 0 {
		return []string{}
	}

	types := make([]string, len(t.InputSources))
	for i, s := range t.InputSources {
		types[i] = s.Source.Type
	}
	return types
}
