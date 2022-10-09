package system

import (
	"flag"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

type Environment string

const (
	// Known environments that are configured in ops/terraform:
	EnvironmentStaging Environment = "staging"
	EnvironmentProd    Environment = "production"
	EnvironmentDev     Environment = "development"
	EnvironmentTest    Environment = "test"
)

func (e Environment) String() string {
	return string(e)
}

func (e Environment) IsKnown() bool {
	switch e {
	case EnvironmentStaging, EnvironmentProd, EnvironmentDev:
		return true
	}
	return false
}

// Cache the environment so we can manipulate it in code for testing:
var env Environment

// Set the global environment cache:
func init() { //nolint:gochecknoinits
	env = Environment(os.Getenv("BACALHAU_ENVIRONMENT"))
	if !env.IsKnown() {
		// Log as debug since we don't want to spam CLI users:
		log.Debug().Msgf("BACALHAU_ENVIRONMENT is not set to a known value: %s", env)

		// This usually happens in the case of a short-lived test cluster, in
		// which case we should default to development. However, we want to
		// avoid using any environment-specific settings for IPFS swarms
		// (which are only configured for production and staging)
		if strings.Contains(os.Args[0], "/_test/") ||
			strings.HasSuffix(os.Args[0], ".test") ||
			flag.Lookup("test.v") != nil {
			env = EnvironmentTest
		} else {
			env = EnvironmentDev
		}
	}
}

func GetEnvironment() Environment {
	return env
}

func IsTest() bool {
	return env == EnvironmentTest
}

func IsStaging() bool {
	return env == EnvironmentStaging
}

func IsProd() bool {
	return env == EnvironmentProd
}

func IsDev() bool {
	return env == EnvironmentDev
}
