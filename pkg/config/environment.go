package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

type Environment string

const (
	// Known environments that are configured in ops/terraform:
	EnvironmentStaging Environment = "staging"
	EnvironmentProd    Environment = "production"
	EnvironmentDev     Environment = "development"
	EnvironmentTest    Environment = "test"
	EnvironmentLocal   Environment = "local"
)

func (e Environment) String() string {
	return string(e)
}

func (e Environment) IsKnown() bool {
	switch e {
	case EnvironmentStaging, EnvironmentProd, EnvironmentDev, EnvironmentTest, EnvironmentLocal:
		return true
	}
	return false
}

func GetConfigEnvironment() Environment {
	env := Environment(os.Getenv("BACALHAU_ENVIRONMENT"))
	if !env.IsKnown() {
		// Log as trace since we don't want to spam CLI users:
		log.Trace().Msgf("BACALHAU_ENVIRONMENT is not set to a known value: %s", env)

		// This usually happens in the case of a short-lived test cluster, in
		// which case we should default to development. However, we want to
		// avoid using any environment-specific settings for IPFS swarms
		// (which are only configured for production and staging)
		if strings.Contains(os.Args[0], "/_test/") ||
			strings.HasSuffix(os.Args[0], ".test") ||
			flag.Lookup("test.v") != nil ||
			flag.Lookup("test.run") != nil {
			env = EnvironmentTest
		} else {
			log.Debug().Msgf("Defaulting to production environment: \n os.Args: %v", os.Args)
			env = EnvironmentProd
		}
	}
	return env
}
