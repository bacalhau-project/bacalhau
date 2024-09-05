package requests

import "github.com/bacalhau-project/bacalhau/pkg/models"

type RegisterRequest struct {
	Info models.NodeInfo
}
type RegisterResponse struct {
	Accepted bool
	Reason   string
}

type UpdateInfoRequest struct {
	Info models.NodeInfo
}

type UpdateInfoResponse struct {
	Accepted bool
	Reason   string
}

type UpdateResourcesRequest struct {
	NodeID            string
	AvailableCapacity models.Resources
	QueueUsedCapacity models.Resources
}

type UpdateResourcesResponse struct{}
