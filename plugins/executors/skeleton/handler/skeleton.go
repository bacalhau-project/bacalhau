package handler

import (
	"bacalhau-exec-skeleton/proto"
	"context"
)

type skeletonHandler struct {
	// supervisor is used to report back state changes for the executor
	supervisor    proto.SupervisorClient
	currentStatus *proto.ExecutionStatus

	// Must embed this structure for unimplemented features
	proto.UnimplementedExecutorServer
}

func NewSkeletonHandler(supervisor proto.SupervisorClient) *skeletonHandler {
	return &skeletonHandler{supervisor: supervisor, currentStatus: proto.ExecutionStatus_Preparing.Enum()}
}

// Run implements proto.ExecutorServer.
func (s *skeletonHandler) Run(ctx context.Context, request *proto.RunRequest) (*proto.RunResponse, error) {
	s.currentStatus = proto.ExecutionStatus_Preparing.Enum()
	s.supervisor.Status(ctx, &proto.StatusRequest{
		Status: s.currentStatus,
	})

	// ...

	s.currentStatus = proto.ExecutionStatus_Completed.Enum()
	s.supervisor.Status(ctx, &proto.StatusRequest{
		Status: s.currentStatus,
	})
	return &proto.RunResponse{ExitCode: 0}, nil
}

// Status will occassionally be called by the supervisor to query the current state of the
// execution and we will return the value that we last set for status.
func (s *skeletonHandler) Status(ctx context.Context, req *proto.StatusRequest) (*proto.StatusResponse, error) {
	return &proto.StatusResponse{
		ExecutionID: req.ExecutionID,
		Status:      *s.currentStatus,
	}, nil
}

// Stop implements proto.ExecutorServer.
func (s *skeletonHandler) Stop(ctx context.Context, request *proto.StopRequest) (*proto.StopResponse, error) {
	return &proto.StopResponse{
		ExecutionID: request.ExecutionID,
	}, nil
}

var _ proto.ExecutorServer = (*skeletonHandler)(nil)
