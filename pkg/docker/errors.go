package docker

import (
	"net/http"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/docker/docker/errdefs"
)

const DockerComponent = "executor/docker"

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

func NewDockerError(err error) *models.BaseError {
	switch {
	case errdefs.IsNotFound(err):
		return handleNotFoundError(err)
	case errdefs.IsConflict(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerConflict).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(DockerComponent)
	case errdefs.IsUnauthorized(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerUnauthorized).
			WithHTTPStatusCode(http.StatusUnauthorized).
			WithComponent(DockerComponent)
	case errdefs.IsForbidden(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerForbidden).
			WithHTTPStatusCode(http.StatusForbidden).
			WithComponent(DockerComponent)
	case errdefs.IsDataLoss(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerDataLoss).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	case errdefs.IsDeadline(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerDeadline).
			WithHTTPStatusCode(http.StatusGatewayTimeout).
			WithComponent(DockerComponent)
	case errdefs.IsCancelled(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerCancelled).
			WithHTTPStatusCode(http.StatusRequestTimeout).
			WithComponent(DockerComponent)
	case errdefs.IsUnavailable(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerUnavailable).
			WithHTTPStatusCode(http.StatusServiceUnavailable).
			WithComponent(DockerComponent)
	case errdefs.IsSystem(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerSystemError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	case errdefs.IsNotImplemented(err):
		return models.NewBaseError(err.Error()).
			WithCode(DockerNotImplemented).
			WithHTTPStatusCode(http.StatusNotImplemented).
			WithComponent(DockerComponent)
	default:
		return models.NewBaseError(err.Error()).
			WithCode(DockerUnknownError).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(DockerComponent)
	}
}

func NewCustomDockerError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(DockerComponent)
}

func handleNotFoundError(err error) *models.BaseError {
	errorLower := strings.ToLower(err.Error())
	if strings.Contains(errorLower, "no such container") {
		return models.NewBaseError(err.Error()).
			WithCode(DockerContainerNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(DockerComponent)
	}
	return models.NewBaseError(err.Error()).
		WithCode(DockerUnknownError).
		WithHTTPStatusCode(http.StatusNotFound).
		WithComponent(DockerComponent)
}
