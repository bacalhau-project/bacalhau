package docker

import (
	"net/http"
	"strings"

	"github.com/docker/docker/errdefs"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const DockerComponent = "Docker"

// Docker-specific error codes
const (
	DockerContainerNotFound = "DockerContainerNotFound"
	DockerImageNotFound     = "DockerImageNotFound"
	DockerNetworkNotFound   = "DockerNetworkNotFound"
	DockerVolumeNotFound    = "DockerVolumeNotFound"
	DockerConflict          = "DockerConflict"
	DockerUnauthorized      = "DockerUnauthorized"
	DockerForbidden         = "DockerForbidden"
	DockerDataLoss          = "DockerDataLoss"
	DockerDeadline          = "DockerDeadline"
	DockerCancelled         = "DockerCancelled"
	DockerUnavailable       = "DockerUnavailable"
	DockerSystemError       = "DockerSystemError"
	DockerNotImplemented    = "DockerNotImplemented"
	DockerUnknownError      = "DockerUnknownError"
)

// Custom Docker error codes
const (
	DockerBridgeNetworkUnattached = "DockerBridgeNetworkUnattached"
	DockerContainerNotRunning     = "DockerContainerNotRunning"
	DockerImageDigestMismatch     = "DockerImageDigestMismatch"
)

func NewDockerError(err error) bacerrors.Error {
	switch {
	case errdefs.IsNotFound(err):
		return handleNotFoundError(err)
	case errdefs.IsConflict(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerConflict).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(DockerComponent)
	case errdefs.IsUnauthorized(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerUnauthorized).
			WithHTTPStatusCode(http.StatusUnauthorized).
			WithComponent(DockerComponent)
	case errdefs.IsForbidden(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerForbidden).
			WithHTTPStatusCode(http.StatusForbidden).
			WithComponent(DockerComponent)
	case errdefs.IsDataLoss(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerDataLoss).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	case errdefs.IsDeadline(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerDeadline).
			WithHTTPStatusCode(http.StatusGatewayTimeout).
			WithComponent(DockerComponent)
	case errdefs.IsCancelled(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerCancelled).
			WithHTTPStatusCode(http.StatusRequestTimeout).
			WithComponent(DockerComponent)
	case errdefs.IsUnavailable(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerUnavailable).
			WithHTTPStatusCode(http.StatusServiceUnavailable).
			WithComponent(DockerComponent)
	case errdefs.IsSystem(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerSystemError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	case errdefs.IsNotImplemented(err):
		return bacerrors.New(err.Error()).
			WithCode(DockerNotImplemented).
			WithHTTPStatusCode(http.StatusNotImplemented).
			WithComponent(DockerComponent)
	default:
		return bacerrors.New(err.Error()).
			WithCode(DockerUnknownError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	}
}

func NewCustomDockerError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New(message).
		WithCode(code).
		WithComponent(DockerComponent)
}

func handleNotFoundError(err error) bacerrors.Error {
	errorLower := strings.ToLower(err.Error())
	if strings.Contains(errorLower, "no such container") {
		return bacerrors.New(err.Error()).
			WithCode(DockerContainerNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(DockerComponent)
	}
	return bacerrors.New(err.Error()).
		WithCode(DockerUnknownError).
		WithHTTPStatusCode(http.StatusNotFound).
		WithComponent(DockerComponent)
}
