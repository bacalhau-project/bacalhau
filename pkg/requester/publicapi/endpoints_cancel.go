package publicapi

import (
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type cancelRequest struct {
	JobID    string `json:"id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}

type CancelRequest = cancelRequest

type cancelResponse struct {
	Job *model.JobWithInfo `json:"job"`
}
