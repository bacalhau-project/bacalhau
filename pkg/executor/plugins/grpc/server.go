package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/plugins/grpc/proto"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	DefaultStreamBufferSize = 1024
)

// TODO: Complete protobuf structure, rather than merely wrapping serialized JSON bytes in protobuf containers.
// Details in: https://github.com/bacalhau-project/bacalhau/issues/2700

type GRPCServer struct {
	Impl executor.Executor

	proto.UnimplementedExecutorServer
}

func (s *GRPCServer) Start(_ context.Context, request *proto.RunCommandRequest) (*proto.StartResponse, error) {
	// NB(forrest): A new context is created for the `Start` operation because `Start` initiates a
	// long-running operation. The context passed as an argument to this method is tied to the gRPC request and is
	// canceled when this method returns. By creating a separate context, we ensure that `Start` has a lifecycle
	// independent of the gRPC request.
	ctx := context.Background()
	args := new(executor.RunCommandRequest)
	if err := json.Unmarshal(request.Params, args); err != nil {
		return nil, err
	}
	if err := s.Impl.Start(ctx, args); err != nil {
		return nil, err
	}
	return &proto.StartResponse{}, nil
}

func (s *GRPCServer) Wait(request *proto.WaitRequest, server proto.Executor_WaitServer) error {
	// NB(forrest): The context obtained from `server.Context()` is appropriate to use here because `Wait`
	// is a streaming RPC. The context remains active for the entire lifetime of the stream and is
	// only canceled when the client or server closes the stream. This behavior is in contrast to
	// unary RPCs (like `Start` and `Run`), where the context is tied to the individual request.
	ctx := server.Context()
	waitC, errC := s.Impl.Wait(ctx, request.GetExecutionID())
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	case res := <-waitC:
		resp, err := json.Marshal(res)
		if err != nil {
			return err
		}
		if err := server.Send(&proto.RunCommandResponse{Params: resp}); err != nil {
			return err
		}
	}
	return nil
}

func (s *GRPCServer) Run(ctx context.Context, request *proto.RunCommandRequest) (*proto.RunCommandResponse, error) {
	args := new(executor.RunCommandRequest)
	if err := json.Unmarshal(request.Params, args); err != nil {
		return nil, err
	}
	result, err := s.Impl.Run(ctx, args)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &proto.RunCommandResponse{Params: b}, nil
}

func (s *GRPCServer) Cancel(ctx context.Context, request *proto.CancelCommandRequest) (*proto.CancelCommandResponse, error) {
	err := s.Impl.Cancel(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}
	return &proto.CancelCommandResponse{}, nil
}

func (s *GRPCServer) IsInstalled(ctx context.Context, _ *proto.IsInstalledRequest) (*proto.IsInstalledResponse, error) {
	installed, err := s.Impl.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	return &proto.IsInstalledResponse{Installed: installed}, nil
}

func (s *GRPCServer) ShouldBid(ctx context.Context, request *proto.ShouldBidRequest) (*proto.ShouldBidResponse, error) {
	var args bidstrategy.BidStrategyRequest
	if err := json.Unmarshal(request.BidRequest, &args); err != nil {
		return nil, err
	}
	result, err := s.Impl.ShouldBid(ctx, args)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &proto.ShouldBidResponse{BidResponse: b}, nil
}

func (s *GRPCServer) ShouldBidBasedOnUsage(
	ctx context.Context,
	request *proto.ShouldBidBasedOnUsageRequest) (*proto.ShouldBidResponse, error) {
	var bidReq bidstrategy.BidStrategyRequest
	if err := json.Unmarshal(request.BidRequest, &bidReq); err != nil {
		return nil, err
	}
	var usage models.Resources
	if err := json.Unmarshal(request.Usage, &usage); err != nil {
		return nil, err
	}
	result, err := s.Impl.ShouldBidBasedOnUsage(ctx, bidReq, usage)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &proto.ShouldBidResponse{BidResponse: b}, nil
}

func (s *GRPCServer) GetOutputStream(request *proto.OutputStreamRequest, server proto.Executor_GetOutputStreamServer) error {
	ctx := server.Context()
	result, err := s.Impl.GetOutputStream(ctx, request.ExecutionID, request.History, request.Follow)
	if err != nil {
		return err
	}
	defer result.Close()

	buffer := make([]byte, DefaultStreamBufferSize)
	for {
		n, err := result.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read data: %w", err)
		}

		res := &proto.OutputStreamResponse{Data: buffer[:n]}
		if err := server.Send(res); err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
	}
	return nil
}
