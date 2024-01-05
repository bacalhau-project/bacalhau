package apimodels

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type GetVersionRequest struct {
	BaseGetRequest
}

type GetVersionResponse struct {
	BaseGetResponse
	Version *models.BuildVersionInfo `json:"Version"`
}
