package requests

import "github.com/bacalhau-project/bacalhau/pkg/models"

type RegisterRequest struct {
	Info models.NodeInfo
}
type RegisterResponse struct {
	Accepted bool
	Error    string
}

type UpdateInfoRequest struct {
	Info models.NodeInfo
}

type UpdateInfoResponse struct {
	Accepted bool
	Error    string
}
