package grpc

import (
	"context"
	"encoding/json"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/plugins/grpc/proto"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TODO: Complete protobuf structure, rather than merely wrapping serialized JSON bytes in protobuf containers.
// Details in: https://github.com/bacalhau-project/bacalhau/issues/2700

var _ (executor.Executor) = (*GRPCClient)(nil)

type GRPCClient struct {
	client proto.ExecutorClient
}

func (c *GRPCClient) IsInstalled(ctx context.Context) (bool, error) {
	resp, err := c.client.IsInstalled(ctx, &proto.IsInstalledRequest{})
	if err != nil {
		return false, err
	}
	return resp.Installed, nil
}

func (c *GRPCClient) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}
	resp, err := c.client.ShouldBid(ctx, &proto.ShouldBidRequest{
		BidRequest: b,
	})
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}
	var out bidstrategy.BidStrategyResponse
	if err := json.Unmarshal(resp.BidResponse, &out); err != nil {
		return bidstrategy.BidStrategyResponse{}, nil
	}
	return out, nil
}

func (c *GRPCClient) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}
	usageBytes, err := json.Marshal(usage)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}
	resp, err := c.client.ShouldBidBasedOnUsage(ctx, &proto.ShouldBidBasedOnUsageRequest{
		BidRequest: reqBytes,
		Usage:      usageBytes,
	})
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, err
	}
	var out bidstrategy.BidStrategyResponse
	if err := json.Unmarshal(resp.BidResponse, &out); err != nil {
		return bidstrategy.BidStrategyResponse{}, nil
	}
	return out, nil
}

func (c *GRPCClient) Run(ctx context.Context, args *executor.RunCommandRequest) (*models.RunCommandResult, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Run(ctx, &proto.RunCommandRequest{Params: b})
	if err != nil {
		return nil, err
	}
	out := new(models.RunCommandResult)
	if err := json.Unmarshal(resp.Params, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *GRPCClient) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	_, err = c.client.Start(ctx, &proto.RunCommandRequest{Params: b})
	if err != nil {
		return err
	}

	return nil
}

func (c *GRPCClient) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	// Create output and error channels
	resultC := make(chan *models.RunCommandResult, 1)
	errC := make(chan error, 1)

	// Initialize the WaitRequest
	waitReq := &proto.WaitRequest{
		ExecutionID: executionID,
	}

	// Make a server-streaming RPC call
	stream, err := c.client.Wait(ctx, waitReq)
	if err != nil {
		errC <- err
		return resultC, errC
	}

	go func() {
		defer close(resultC)
		defer close(errC)

		// block until we receive a message from the stream or an error.
		resp, err := stream.Recv()
		if err != nil {
			errC <- err
			return
		}

		// Convert proto.WaitResponse to models.RunCommandResult
		out := new(models.RunCommandResult)
		if err := json.Unmarshal(resp.Params, out); err != nil {
			errC <- err
			return
		}

		// Send the result to the channel
		resultC <- out
	}()

	return resultC, errC
}

func (c *GRPCClient) Cancel(ctx context.Context, id string) error {
	_, err := c.client.Cancel(ctx, &proto.CancelCommandRequest{ExecutionID: id})
	if err != nil {
		return err
	}
	return nil
}

func (c *GRPCClient) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	respStream, err := c.client.GetOutputStream(ctx, &proto.OutputStreamRequest{
		ExecutionID: executionID,
		History:     withHistory,
		Follow:      follow,
	})
	if err != nil {
		return nil, err
	}

	return &StreamReader{stream: respStream}, nil
}

type StreamReader struct {
	stream proto.Executor_GetOutputStreamClient
	buffer []byte
}

func (sr *StreamReader) Read(p []byte) (n int, err error) {
	if len(sr.buffer) == 0 { // if buffer is empty, fill it by reading from the stream
		response, err := sr.stream.Recv()
		if err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}
		sr.buffer = response.Data
	}

	n = copy(p, sr.buffer)    // copy from buffer to p
	sr.buffer = sr.buffer[n:] // update buffer

	return n, nil
}

func (sr *StreamReader) Close() error {
	return sr.stream.CloseSend()
}
