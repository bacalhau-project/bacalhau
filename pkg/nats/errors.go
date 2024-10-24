package nats

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const transportComponent = "Transport"
const transportServerComponent = "TransportServer"
const transportClientComponent = "TransportClient"

// NewConfigurationError creates a new error for a configuration error
func NewConfigurationError(message string, args ...interface{}) bacerrors.Error {
	return bacerrors.New(message, args...).
		WithComponent(transportComponent).
		WithCode(bacerrors.ConfigurationError)
}

// NewConfigurationWrappedError creates a new error for a configuration error
func NewConfigurationWrappedError(err error, message string, args ...interface{}) bacerrors.Error {
	return bacerrors.Wrap(err, message, args...).
		WithComponent(transportComponent).
		WithCode(bacerrors.ConfigurationError)
}

func interceptConnectionError(err error, servers string) error {
	switch {
	case errors.Is(err, nats.ErrNoServers):
		defaultServers := strings.Join(config.Default.Compute.Orchestrators, ",")
		hint := fmt.Sprintf(`to resolve this, either:
1. Ensure that the orchestrator is running and reachable at %s
2. Update the configuration to use a different orchestrator address using:
   a. The '-c %s=<new_address>' flag with your serve command
   b. Set the address in a configuration file with '%s config set %s=<new_address>'`,
			servers, types.ComputeOrchestratorsKey, os.Args[0], types.ComputeOrchestratorsKey)

		if servers == defaultServers {
			hint += `
3. If you are trying to connect to the demo network, use 'bootstrap.demo.bacalhau.org:4222' as your address`
		}

		return bacerrors.New("no orchestrator available for connection at %s", servers).
			WithComponent(transportClientComponent).
			WithCode(bacerrors.ConfigurationError).
			WithHint(hint)
	default:
		return bacerrors.Wrap(err, "failed to connect to %s", servers).
			WithComponent(transportClientComponent).
			WithCode(bacerrors.ConfigurationError)
	}
}
