package env

import "github.com/bacalhau-project/bacalhau/pkg/bacerrors"

// Error codes for environment variable resolution
const (
	errComponent = "EnvResolver"
)

func newErrNotAllowed(name string) bacerrors.Error {
	return bacerrors.Newf("environment variable '%s' is not in allowed patterns", name).
		WithCode(bacerrors.UnauthorizedError).
		WithComponent(errComponent).
		WithHint("Check allowed patterns of the compute node's configuration")
}

func newErrNotFound(name string) bacerrors.Error {
	return bacerrors.Newf("required environment variable '%s' not found", name).
		WithCode(bacerrors.NotFoundError).
		WithComponent(errComponent).
		WithHint("Check the host environment variables")
}
