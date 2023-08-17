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

// TODO: Complete protobuf structure, rather than merely wrapping serialized JSON bytes in protobuf containers.
// Details in: https://github.com/bacalhau-project/bacalhau/issues/2700

type GRPCServer struct {
	Impl executor.Executor
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

func (s *GRPCServer) ShouldBidBasedOnUsage(ctx context.Context, request *proto.ShouldBidBasedOnUsageRequest) (*proto.ShouldBidResponse, error) {
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

func (s *GRPCServer) GetOutputStream(request *proto.OutputStreamRequest, srv proto.Executor_GetOutputStreamServer) error {
	result, err := s.Impl.GetOutputStream(context.TODO(), request.ExecutionID, request.History, request.Follow)
	if err != nil {
		return err
	}
	defer result.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := result.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read data: %w", err)
		}

		res := &proto.OutputStreamResponse{Data: buffer[:n]}
		if err := srv.Send(res); err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
	}
	return nil
}
