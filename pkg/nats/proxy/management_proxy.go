package proxy

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

type managementRequest interface {
	requests.RegisterRequest | requests.UpdateInfoRequest
}

type managementResponse interface {
	requests.RegisterResponse | requests.UpdateInfoResponse
}

type ManagementProxyParams struct {
	Conn *nats.Conn
}

// type ManagementProxy is a proxy for a compute node to register itself with a requester node.
type ManagementProxy struct {
	conn *nats.Conn
}

// NewRegistrationProxy creates a new RegistrationProxy for the local compute node
// bound to a provided NATS connection.
func NewManagementProxy(params ManagementProxyParams) *ManagementProxy {
	return &ManagementProxy{
		conn: params.Conn,
	}
}

// Register sends a `requester.RegisterInfoRequest` containing the current compute node's
// NodeID to the requester node.
func (p *ManagementProxy) Register(ctx context.Context,
	request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	var err error
	var response *requests.RegisterResponse

	response, err = send[requests.RegisterRequest, requests.RegisterResponse](
		ctx, p.conn, request.Info.NodeID, request, RegisterNode)

	// TODO(ross): This is not always true
	response.Accepted = true

	return response, err
}

func send[Q managementRequest, R managementResponse](
	ctx context.Context,
	conn *nats.Conn,
	nodeID string,
	req Q, method string) (*R, error) {
	data, err := json.Marshal(req)
	if err != nil {
		log.Ctx(ctx).Error().Err(errors.WithStack(err)).Msgf("%s: failed to marshal request", reflect.TypeOf(req))
		return nil, err
	}

	subject := managementPublishSubject(nodeID, method)
	log.Ctx(ctx).Trace().Msgf("Sending request to subject %s", subject)

	if err = conn.Publish(subject, data); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error sending request to subject %s", subject)
		return nil, err
	}

	// TODO: Read/parse response and return it
	response := R{}
	return &response, nil
}

// UpdateInfo sends the latest node info from the current compute node to the server.
// We will do this even if we are not registered so that we will generate a regular
// error explaining why the update failed.
func (p *ManagementProxy) UpdateInfo(ctx context.Context,
	request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	var err error
	var response *requests.UpdateInfoResponse

	response, err = send[requests.UpdateInfoRequest, requests.UpdateInfoResponse](
		ctx, p.conn, request.Info.NodeID, request, UpdateNodeInfo)

	// TODO(ross): This is not always true
	response.Accepted = true

	return response, err
}
