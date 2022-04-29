package scheduler

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
)

// a useful generic scheduler that given a function to write a job event
// will look after a lot of the boilerplate on behalf on a scheduler implementation

type GenericScheduler struct {
	NodeId string
	Jobs   map[string]*types.Job
	// the list of functions to call when we get an update about a job
	SubscribeFuncs    []func(jobEvent *types.JobEvent, job *types.Job)
	WriteEventHandler func(event *types.JobEvent) error
}

func NewGenericScheduler(
	nodeId string,
	writeEvent func(event *types.JobEvent) error,
) *GenericScheduler {
	return &GenericScheduler{
		NodeId:            nodeId,
		Jobs:              make(map[string]*types.Job),
		SubscribeFuncs:    []func(jobEvent *types.JobEvent, job *types.Job){},
		WriteEventHandler: writeEvent,
	}
}

func (scheduler *GenericScheduler) writeEvent(event *types.JobEvent) error {
	if event.NodeId == "" {
		event.NodeId = scheduler.NodeId
	}
	return scheduler.WriteEventHandler(event)
}

func (scheduler *GenericScheduler) ReadEvent(event *types.JobEvent) {
	// let's initialise the state for this job because it was just created
	if event.EventName == system.JOB_EVENT_CREATED {
		scheduler.Jobs[event.JobId] = &types.Job{
			Id:    event.JobId,
			Owner: event.NodeId,
			Spec:  nil,
			Deal:  nil,
			State: make(map[string]*types.JobState),
		}
	}

	// for "create" and "update" events - this will be filled in
	if event.JobSpec != nil {
		scheduler.Jobs[event.JobId].Spec = event.JobSpec
	}

	// only the owner of the job can update
	if event.JobDeal != nil {
		scheduler.Jobs[event.JobId].Deal = event.JobDeal
	}

	// both the jobState struct and the NodeId are required
	// because the job state is "against" the node
	if event.JobState != nil && event.NodeId != "" {
		scheduler.Jobs[event.JobId].State[event.NodeId] = event.JobState
	}

	for _, subscribeFunc := range scheduler.SubscribeFuncs {
		go subscribeFunc(event, scheduler.Jobs[event.JobId])
	}
}

func (scheduler *GenericScheduler) List() (types.ListResponse, error) {
	return types.ListResponse{
		Jobs: scheduler.Jobs,
	}, nil
}

func (scheduler *GenericScheduler) Get(id string) (*types.Job, error) {
	job, ok := scheduler.Jobs[id]
	if !ok {
		return nil, fmt.Errorf("Job %s not found", id)
	} else {
		return job, nil
	}
}

func (scheduler *GenericScheduler) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	scheduler.SubscribeFuncs = append(scheduler.SubscribeFuncs, subscribeFunc)
}

func (scheduler *GenericScheduler) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	jobUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("Error in creating job id. %s", err)
	}

	jobId := jobUuid.String()

	err = scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_CREATED,
		JobSpec:   spec,
		JobDeal:   deal,
	})

	if err != nil {
		return nil, err
	}

	job := &types.Job{
		Id:    jobId,
		Spec:  spec,
		Deal:  deal,
		State: make(map[string]*types.JobState),
	}

	return job, nil
}

func (scheduler *GenericScheduler) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_DEAL_UPDATED,
		JobDeal:   deal,
	})
}

func (scheduler *GenericScheduler) AcceptJobBid(jobId, nodeId string) error {
	job, err := scheduler.Get(jobId)
	if err != nil {
		return err
	}
	job.Deal.AssignedNodes = append(job.Deal.AssignedNodes, nodeId)
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_BID_ACCEPTED,
		JobDeal:   job.Deal,
		JobState: &types.JobState{
			State: system.JOB_STATE_RUNNING,
		},
	})
}

func (scheduler *GenericScheduler) RejectJobBid(jobId, nodeId, message string) error {
	if message == "" {
		message = "Job bid rejected by client"
	}
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_BID_REJECTED,
		JobState: &types.JobState{
			State:  system.JOB_STATE_BID_REJECTED,
			Status: message,
		},
	})
}

func (scheduler *GenericScheduler) AcceptResult(jobId, nodeId string) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_RESULTS_ACCEPTED,
		JobState: &types.JobState{
			State: system.JOB_STATE_RESULTS_ACCEPTED,
		},
	})
}

func (scheduler *GenericScheduler) RejectResult(jobId, nodeId, message string) error {
	if message == "" {
		message = "Job result rejected by client"
	}
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_RESULTS_REJECTED,
		JobState: &types.JobState{
			State:  system.JOB_STATE_RESULTS_REJECTED,
			Status: message,
		},
	})
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (scheduler *GenericScheduler) BidJob(jobId string) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_BID,
		JobState: &types.JobState{
			State: system.JOB_STATE_BIDDING,
		},
	})
}

func (scheduler *GenericScheduler) SubmitResult(jobId, status string, outputs []types.StorageSpec) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_RESULTS,
		JobState: &types.JobState{
			State:   system.JOB_STATE_COMPLETE,
			Status:  status,
			Outputs: outputs,
		},
	})
}

func (scheduler *GenericScheduler) ErrorJob(jobId, status string) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_ERROR,
		JobState: &types.JobState{
			State:  system.JOB_STATE_ERROR,
			Status: status,
		},
	})
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (scheduler *GenericScheduler) ErrorJobForNode(jobId, nodeId, status string) error {
	return scheduler.writeEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_ERROR,
		JobState: &types.JobState{
			State:  system.JOB_STATE_ERROR,
			Status: status,
		},
	})
}
