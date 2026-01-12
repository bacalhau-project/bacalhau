package docker

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/errdefs"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
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
	case errdefs.IsNotFound(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsNotFound
		return handleNotFoundError(err)
	case errdefs.IsConflict(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsConflict
		return bacerrors.Newf("%s", err).
			WithCode(Conflict).
			WithHTTPStatusCode(http.StatusConflict)
	case errdefs.IsUnauthorized(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsUnauthorized
		return bacerrors.Newf("%s", err).
			WithCode(Unauthorized).
			WithHTTPStatusCode(http.StatusUnauthorized).
			WithHint("Ensure you have the necessary permissions and that your credentials are correct. " +
				"You may need to log in to Docker again.")
	case errdefs.IsForbidden(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsPermissionDenied
		return bacerrors.Newf("%s", err).
			WithCode(Forbidden).
			WithHTTPStatusCode(http.StatusForbidden).
			WithHint(fmt.Sprintf("You don't have permission to perform this action. "+
				"Supply the node with valid Docker login credentials using the %s and %s environment variables",
				UsernameEnvVar, PasswordEnvVar))
	case errdefs.IsDataLoss(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsDataLoss
		return bacerrors.Newf("%s", err).
			WithCode(DataLoss).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithFailsExecution()
	case errdefs.IsDeadline(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsDeadlineExceeded
		return bacerrors.Newf("%s", err).
			WithCode(Deadline).
			WithHTTPStatusCode(http.StatusGatewayTimeout).
			WithHint("The operation timed out. This could be due to network issues or high system load. " +
				"Try again later or check your network connection.").
			WithRetryable()
	case errdefs.IsCancelled(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsCanceled
		return bacerrors.Newf("%s", err).
			WithCode(Cancelled).
			WithHTTPStatusCode(http.StatusRequestTimeout).
			WithHint("The operation was cancelled. " +
				"This is often due to user intervention or a competing operation.")
	case errdefs.IsUnavailable(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsUnavailable
		return bacerrors.Newf("%s", err).
			WithCode(Unavailable).
			WithHTTPStatusCode(http.StatusServiceUnavailable).
			WithHint("The Docker daemon or a required service is unavailable. " +
				"Check if the Docker daemon is running and healthy.").
			WithRetryable()
	case errdefs.IsSystem(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsInternal
		return bacerrors.Newf("%s", err).
			WithCode(SystemError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithHint("An internal system error occurred. This could be due to resource constraints. " +
				"Check system resources and Docker logs for more information.").
			WithFailsExecution()
	case errdefs.IsNotImplemented(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsNotImplemented
		return bacerrors.Newf("%s", err).
			WithCode(NotImplemented).
			WithHTTPStatusCode(http.StatusNotImplemented).
			WithHint("This feature is not implemented in your version of Docker. " +
				"Check Docker documentation for feature availability and consider upgrading if necessary.")
	default:
		return bacerrors.Newf("%s", err).
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
	case errdefs.IsNotFound(err) || errdefs.IsForbidden(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs
		return bacerrors.Newf("image not available: %q", image).
			WithHint(fmt.Sprintf(`To resolve this, either:
1. Check if the image exists in the registry and the name is correct
2. If the image is private, supply the node with valid Docker login credentials using the %s and %s environment variables`,
				UsernameEnvVar, PasswordEnvVar)).
			WithCode(ImageNotFound)
	case errdefs.IsInvalidParameter(err): //nolint:staticcheck // TODO: migrate to containerd cerrdefs.IsInvalidArgument
		return bacerrors.Newf("invalid image format: %q", image).
			WithHint("Ensure the image name is valid and the image is available in the registry").
			WithCode(ImageInvalid)
	default:
		return NewDockerError(err)
	}
}

func NewCustomDockerError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.Newf("%s", message).
		WithCode(code).
		WithComponent(Component)
}

func handleNotFoundError(err error) bacerrors.Error {
	errorLower := strings.ToLower(err.Error())
	if strings.Contains(errorLower, "no such container") {
		return bacerrors.Newf("%s", err).
			WithCode(ContainerNotFound).
			WithHTTPStatusCode(http.StatusNotFound)
	}
	return bacerrors.Newf("%s", err).
		WithCode(NotFound).
		WithHTTPStatusCode(http.StatusNotFound)
}
