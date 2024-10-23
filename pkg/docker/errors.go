package docker

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/errdefs"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
)

const Component = "Docker"

// Docker-specific error codes
const (
	ContainerNotFound = "ContainerNotFound"
	ImageNotFound     = "ImageNotFound"
	ImageInvalid      = "ImageInvalid"
	NotFound          = "NotFound"
	Conflict          = "Conflict"
	Unauthorized      = "Unauthorized"
	Forbidden         = "Forbidden"
	DataLoss          = "DataLoss"
	Deadline          = "Deadline"
	Cancelled         = "Cancelled"
	Unavailable       = "Unavailable"
	SystemError       = "SystemError"
	NotImplemented    = "NotImplemented"
	UnknownError      = "UnknownError"
)

// Custom Docker error codes
const (
	BridgeNetworkUnattached = "BridgeNetworkUnattached"
	ContainerNotRunning     = "ContainerNotRunning"
	ImageDigestMismatch     = "ImageDigestMismatch"
)

func NewDockerError(err error) (bacErr bacerrors.Error) {
	defer func() {
		if bacErr != nil {
			bacErr = bacErr.WithComponent(Component)
		}
	}()
	switch {
	case errdefs.IsNotFound(err):
		return handleNotFoundError(err)
	case errdefs.IsConflict(err):
		return bacerrors.New("%s", err).
			WithCode(Conflict).
			WithHTTPStatusCode(http.StatusConflict)
	case errdefs.IsUnauthorized(err):
		return bacerrors.New("%s", err).
			WithCode(Unauthorized).
			WithHTTPStatusCode(http.StatusUnauthorized).
			WithHint("Ensure you have the necessary permissions and that your credentials are correct. " +
				"You may need to log in to Docker again.")
	case errdefs.IsForbidden(err):
		return bacerrors.New("%s", err).
			WithCode(Forbidden).
			WithHTTPStatusCode(http.StatusForbidden).
			WithHint(fmt.Sprintf("You don't have permission to perform this action. "+
				"Supply the node with valid Docker login credentials using the %s and %s environment variables",
				config_legacy.DockerUsernameEnvVar, config_legacy.DockerPasswordEnvVar))
	case errdefs.IsDataLoss(err):
		return bacerrors.New("%s", err).
			WithCode(DataLoss).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithFailsExecution()
	case errdefs.IsDeadline(err):
		return bacerrors.New("%s", err).
			WithCode(Deadline).
			WithHTTPStatusCode(http.StatusGatewayTimeout).
			WithHint("The operation timed out. This could be due to network issues or high system load. " +
				"Try again later or check your network connection.").
			WithRetryable()
	case errdefs.IsCancelled(err):
		return bacerrors.New("%s", err).
			WithCode(Cancelled).
			WithHTTPStatusCode(http.StatusRequestTimeout).
			WithHint("The operation was cancelled. " +
				"This is often due to user intervention or a competing operation.")
	case errdefs.IsUnavailable(err):
		return bacerrors.New("%s", err).
			WithCode(Unavailable).
			WithHTTPStatusCode(http.StatusServiceUnavailable).
			WithHint("The Docker daemon or a required service is unavailable. " +
				"Check if the Docker daemon is running and healthy.").
			WithRetryable()
	case errdefs.IsSystem(err):
		return bacerrors.New("%s", err).
			WithCode(SystemError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithHint("An internal system error occurred. This could be due to resource constraints. " +
				"Check system resources and Docker logs for more information.").
			WithFailsExecution()
	case errdefs.IsNotImplemented(err):
		return bacerrors.New("%s", err).
			WithCode(NotImplemented).
			WithHTTPStatusCode(http.StatusNotImplemented).
			WithHint("This feature is not implemented in your version of Docker. " +
				"Check Docker documentation for feature availability and consider upgrading if necessary.")
	default:
		return bacerrors.New("%s", err).
			WithCode(UnknownError).
			WithHTTPStatusCode(http.StatusInternalServerError)
	}
}

func NewDockerImageError(err error, image string) (bacErr bacerrors.Error) {
	defer func() {
		if bacErr != nil {
			bacErr = bacErr.
				WithComponent(Component).
				WithDetail("Image", image)
		}
	}()

	switch {
	case errdefs.IsNotFound(err) || errdefs.IsForbidden(err):
		return bacerrors.New("image not available: %q", image).
			WithHint(fmt.Sprintf(`To resolve this, either:
1. Check if the image exists in the registry and the name is correct
2. If the image is private, supply the node with valid Docker login credentials using the %s and %s environment variables`,
				config_legacy.DockerUsernameEnvVar, config_legacy.DockerPasswordEnvVar)).
			WithCode(ImageNotFound)
	case errdefs.IsInvalidParameter(err):
		return bacerrors.New("invalid image format: %q", image).
			WithHint("Ensure the image name is valid and the image is available in the registry").
			WithCode(ImageInvalid)
	default:
		return NewDockerError(err)
	}
}

func NewCustomDockerError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(code).
		WithComponent(Component)
}

func handleNotFoundError(err error) bacerrors.Error {
	errorLower := strings.ToLower(err.Error())
	if strings.Contains(errorLower, "no such container") {
		return bacerrors.New("%s", err).
			WithCode(ContainerNotFound).
			WithHTTPStatusCode(http.StatusNotFound)
	}
	return bacerrors.New("%s", err).
		WithCode(NotFound).
		WithHTTPStatusCode(http.StatusNotFound)
}
