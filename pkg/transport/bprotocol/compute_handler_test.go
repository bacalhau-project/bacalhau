//go:build unit || !integration

package bprotocol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ComputeProxyTestSuite struct {
	suite.Suite
	ctx     context.Context
	handler *ComputeHandler
	proxy   *ComputeProxy
}

func TestComputeProxyTestSuite(t *testing.T) {
	suite.Run(t, new(ComputeProxyTestSuite))
}

func (s *ComputeProxyTestSuite) SetupSuite() {
	s.ctx = context.Background()

	computeNode, err := libp2p.NewHostForTest(s.ctx)
	require.NoError(s.T(), err)

	proxyNode, err := libp2p.NewHostForTest(s.ctx, computeNode)
	require.NoError(s.T(), err)

	s.handler = NewComputeHandler(ComputeHandlerParams{
		Host:            computeNode,
		ComputeEndpoint: &TestEndpoint{},
	})
	s.proxy = NewComputeProxy(ComputeProxyParams{
		Host: proxyNode,
	})
}

type TestEndpoint struct{}

func (t *TestEndpoint) AskForBid(context.Context, compute.AskForBidRequest) (compute.AskForBidResponse, error) {
	return compute.AskForBidResponse{}, errors.New("error raised by AskForBid")
}
func (t *TestEndpoint) BidAccepted(context.Context, compute.BidAcceptedRequest) (compute.BidAcceptedResponse, error) {
	return compute.BidAcceptedResponse{ExecutionMetadata: compute.ExecutionMetadata{ExecutionID: "test"}}, nil
}
func (t *TestEndpoint) BidRejected(context.Context, compute.BidRejectedRequest) (compute.BidRejectedResponse, error) {
	return compute.BidRejectedResponse{}, errors.New("No test implementation")
}
func (t *TestEndpoint) CancelExecution(context.Context, compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	return compute.CancelExecutionResponse{}, errors.New("No test implementation")
}
func (t *TestEndpoint) ExecutionLogs(ctx context.Context, request compute.ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return nil, errors.New("No test implementation")
}

func (s *ComputeProxyTestSuite) TeardownSuite() {
	s.proxy.host.Close()
}

// Gets the metadata for calling the compute node of the test
func (s *ComputeProxyTestSuite) getRoutingMetadataForCompute() compute.RoutingMetadata {
	return compute.RoutingMetadata{
		SourcePeerID: s.proxy.host.ID().String(),
		TargetPeerID: s.handler.host.ID().String(),
	}
}

func (s *ComputeProxyTestSuite) TestSimpleError() {
	_, err := s.proxy.AskForBid(s.ctx, compute.AskForBidRequest{
		RoutingMetadata: s.getRoutingMetadataForCompute(),
	})

	require.Error(s.T(), err)
	require.Equal(s.T(), "error raised by AskForBid", err.Error())
}

func (s *ComputeProxyTestSuite) TestSimpleSuccess() {
	response, err := s.proxy.BidAccepted(s.ctx, compute.BidAcceptedRequest{
		RoutingMetadata: s.getRoutingMetadataForCompute(),
	})

	// Expect a BidAcceptedResponse, err result.

	require.NoError(s.T(), err)
	require.Equal(s.T(), "test", response.ExecutionID)
}
