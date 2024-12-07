package proxy

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

const (
	requestTimeout = 2 * time.Second
)

type managementRequest interface {
	legacy.RegisterRequest | legacy.UpdateInfoRequest | legacy.UpdateResourcesRequest
}

type managementResponse interface {
	legacy.RegisterResponse | legacy.UpdateInfoResponse | legacy.UpdateResourcesResponse
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
// NodeID to the requester node. It uses NATS request-reply to get a response and returns
// the response to the caller.
func (p *ManagementProxy) Register(ctx context.Context,
	request legacy.RegisterRequest) (*legacy.RegisterResponse, error) {
	var err error
	var asyncRes *concurrency.AsyncResult[legacy.RegisterResponse]

	asyncRes, err = send[legacy.RegisterRequest, legacy.RegisterResponse](
		ctx, p.conn, request.Info.NodeID, request, RegisterNode)

	if err != nil {
		return nil, errors.Wrap(err, "failed to send response to registration request")
	}

	return &asyncRes.Value, asyncRes.Err
}

// UpdateInfo sends the latest node info from the current compute node to the server.
// We will do this even if we are not registered so that we will generate a regular
// error explaining why the update failed.
func (p *ManagementProxy) UpdateInfo(ctx context.Context,
	request legacy.UpdateInfoRequest) (*legacy.UpdateInfoResponse, error) {
	var err error
	var asyncRes *concurrency.AsyncResult[legacy.UpdateInfoResponse]

	asyncRes, err = send[legacy.UpdateInfoRequest, legacy.UpdateInfoResponse](
		ctx, p.conn, request.Info.NodeID, request, UpdateNodeInfo)

	if err != nil {
		return nil, errors.Wrap(err, "failed to send response to update info request")
	}

	return &asyncRes.Value, asyncRes.Err
}

// UpdateResources sends the currently available resources from the current compute
// node to the server.
func (p *ManagementProxy) UpdateResources(ctx context.Context,
	request legacy.UpdateResourcesRequest) (*legacy.UpdateResourcesResponse, error) {
	var err error
	var asyncRes *concurrency.AsyncResult[legacy.UpdateResourcesResponse]

	asyncRes, err = send[legacy.UpdateResourcesRequest, legacy.UpdateResourcesResponse](
		ctx, p.conn, request.NodeID, request, UpdateResources)

	if err != nil {
		return nil, errors.Wrap(err, "failed to send response to update resources request")
	}

	return &asyncRes.Value, asyncRes.Err
}

// send will deliver a message to the requester node and wait for a response. The response
// will be wrapped in a concurrency.AsyncResult and returned to the caller. It is the caller's
// responsibility to unwrap the asyncresult and check the error.
func send[Q managementRequest, R managementResponse](
	ctx context.Context,
	conn *nats.Conn,
	nodeID string,
	req Q, method string) (*concurrency.AsyncResult[R], error) {
	data, err := json.Marshal(req)
	if err != nil {
		log.Ctx(ctx).Error().Err(errors.WithStack(err)).Msgf("%s: failed to marshal request", reflect.TypeOf(req))
		return nil, err
	}

	subject := managementPublishSubject(nodeID, method)
	log.Ctx(ctx).Trace().Msgf("Sending %T request to subject %s", req, subject)

	respMsg, err := conn.Request(subject, data, requestTimeout)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("error sending request to subject %s", subject)
		return nil, err
	}

	var response concurrency.AsyncResult[R]
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response to NATS request")
	}

	return &response, nil
}
