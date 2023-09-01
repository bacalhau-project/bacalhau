package apimodels

import "github.com/bacalhau-project/bacalhau/pkg/models"

type VersionRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}

type VersionResponse struct {
	VersionInfo *models.BuildVersionInfo `json:"build_version_info"`
}
