package nats

import (
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
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
