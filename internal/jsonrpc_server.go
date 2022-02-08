package internal

import (
	"github.com/filecoin-project/bacalhau/internal/types"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type SubmitArgs struct {
	Job *types.Job
}


func (server *JobServer) Submit(args *SubmitArgs, reply *types.Job) error {
	err := server.ComputeNode.Publish(args.Job)
	
	if err != nil {
		return err
	}
	*reply = *args.Job
	return nil
}
