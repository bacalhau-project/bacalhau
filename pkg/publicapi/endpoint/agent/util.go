package agent

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// RedactSensitiveInfo redacts sensitive information from the auth config
func redactConfigSensitiveInfo(config types.Bacalhau) (types.Bacalhau, error) {
	deepCopyOfConfig, err := config.Copy()
	if err != nil {
		return types.Bacalhau{}, err
	}

	const redactedMask = "********"
	if deepCopyOfConfig.Compute.Auth.Token != "" {
		deepCopyOfConfig.Compute.Auth.Token = redactedMask
	}
	if deepCopyOfConfig.Orchestrator.Auth.Token != "" {
		deepCopyOfConfig.Orchestrator.Auth.Token = redactedMask
	}

	// Redact user passwords and API keys
	for userIdx := range deepCopyOfConfig.API.Auth.Users {
		user := &deepCopyOfConfig.API.Auth.Users[userIdx]
		if user.Password != "" {
			user.Password = redactedMask
		}
		if user.APIKey != "" {
			user.APIKey = redactedMask
		}
	}

	return deepCopyOfConfig, nil
}
