package requesternode

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/exp/maps"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	sync "github.com/lukemarsden/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
)

// types of actions that can be performed on a shard state machine
type shardStateAction int

const (
	// bid received from a compute node.
	actionBidReceived shardStateAction = iota // must be first

	// result received from a compute node.
	actionResultReceived

	// result published after it has been verified
	actionResultsPublished
)

func (a shardStateAction) String() string {
	return [...]string{"ActionBidReceived", "ActionResultReceived", "ActionResultsPublished", "ActionFail"}[a]
}

// request to change the state of the fsm
type shardStateRequest struct {
	action       shardStateAction
	sourceNodeID string // optional field indicating the node that triggered the request
}

// types of shard state machines
type shardStateType int

const (
	shardInitialState shardStateType = iota // must be first

	// Shard is enqueuing bids waiting Min bids before start accepting/rejecting bids.
	shardEnqueuingBids

	// Min bids have been received, shard is now selecting from the previously enqueued bids.
	shardSelectingBids

	// Shard is still accepting bids waiting to reach concurrency limit.
	shardAcceptingBids

	// Shard is waiting on compute nodes to submit their result proposals.
	shardWaitingForResults

	// All compute nodes submitted their results proposal, and now verifier is verifying them.
	shardVerifyingResults

	// Verifier has verified the results, and now the shard is publishing the results to the requester.
	shardWaitingToPublishResults

	// The job has failed due to an error.
	shardError

	// The job has been completed, either successfully, or due to an error.
	shardCompleted
)

func (s shardStateType) String() string {
	return [...]string{
		"InitialState", "EnqueuingBids", "SelectingBids", "AcceptingBids", "WaitingForResults",
		"VerifyingResults", "WaitingToPublishResults", "Error", "Completed"}[s]
}

type shardStateMachineManager struct {
	// map fo the job ID and job state machine.
	// Used to find the job state machine for a given ID.
	shardStates map[string]*shardStateMachine
	mu          sync.Mutex
}

func newShardStateMachineManager() *shardStateMachineManager {
	stateManager := &shardStateMachineManager{
		shardStates: make(map[string]*shardStateMachine),
	}

	stateManager.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "RequesterNode.ShardStateMachineManagerMu",
	})

	return stateManager
}

// Start a state machine for all the shards in the job, if they don't exit already
func (m *shardStateMachineManager) startShardsState(
	ctx context.Context, job *model.Job, n *RequesterNode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// explode the job into shard states
	for i := 0; i < job.ExecutionPlan.TotalShards; i++ {
		shard := model.JobShard{Job: job, Index: i}
		if _, ok := m.shardStates[shard.ID()]; !ok {
			shardState := newShardStateMachine(ctx, shard, n)
			m.shardStates[shard.ID()] = shardState

			go func() {
				shardState.run(ctx)
			}()
		} // else, fsm was already running
	}
}

func (m *shardStateMachineManager) GetShardState(shard model.JobShard) (*shardStateMachine, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	shardFsm, ok := m.shardStates[shard.ID()]
	return shardFsm, ok
}

type shardStateMachine struct {
	shard model.JobShard
	node  *RequesterNode
	req   chan shardStateRequest

	currentState  shardStateType
	previousState shardStateType
	errorMsg      string

	// keep track of nodes that have already bid on this shard to deduplicate bids and only accept results
	// from nodes that have an accepted bid.
	biddingNodes map[string]struct{}

	// keep track of nodes that have already submitted their result proposals to deduplicate results, and know when
	// result verification should start.
	completedNodes map[string]struct{}
}

func newShardStateMachine(ctx context.Context, shard model.JobShard, node *RequesterNode) *shardStateMachine {
	return &shardStateMachine{
		shard:          shard,
		node:           node,
		req:            make(chan shardStateRequest),
		currentState:   shardInitialState,
		biddingNodes:   make(map[string]struct{}),
		completedNodes: make(map[string]struct{}),
	}
}

func (m *shardStateMachine) String() string {
	return fmt.Sprintf("[%s] shard: %s at state: %s", m.node.ID[:8], m.shard, m.currentState)
}

// run the state machine until it is completed.
func (m *shardStateMachine) run(ctx context.Context) {
	for state := enqueuedState; state != nil; {
		// TODO: #559 Should we create a new context and span for each state execution?
		state = state(ctx, m)
	}
	// close the request channel.
	// Check `sendRequest` comments for more details.
	close(m.req)
}

func (m *shardStateMachine) bid(ctx context.Context, sourceNodeID string) {
	m.sendRequest(ctx, shardStateRequest{action: actionBidReceived, sourceNodeID: sourceNodeID})
}

func (m *shardStateMachine) verifyResult(ctx context.Context, sourceNodeID string) {
	m.sendRequest(ctx, shardStateRequest{action: actionResultReceived, sourceNodeID: sourceNodeID})
}

func (m *shardStateMachine) resultsPublished(ctx context.Context, sourceNodeID string) {
	m.sendRequest(ctx, shardStateRequest{action: actionResultsPublished, sourceNodeID: sourceNodeID})
}

// send a request to the state machine by enqueuing it in the request channel.
// it is possible due to race condition or duplicate network events that a
// request is sent after the fsm is completed and no longer a goroutin is
// consuming from the channel, which will lead to a deadlock in the
// requesternode when trying to send the request.
// To mitigate this, we close the channel when the fsm is completed, and handle
// the panic gracefully here.
func (m *shardStateMachine) sendRequest(ctx context.Context, request shardStateRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Ctx(ctx).Warn().Msgf("%s ignoring action after channel closed: %s", m, request.action)
			go func() {

			}()
		}
	}()
	m.req <- request
}

// ------------------------------------
// Shard State Machine Functions
// ------------------------------------
type stateFn func(context.Context, *shardStateMachine) stateFn

func (m *shardStateMachine) transitionedTo(ctx context.Context, newState shardStateType) {
	log.Ctx(ctx).Debug().Msgf("%s transitioning from %s -> %s", m, m.currentState, newState)
	m.previousState = m.currentState
	m.currentState = newState
}

// Shard is enqueuing bids waiting Min bids before start accepting/rejecting bids.
func enqueuedState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardEnqueuingBids)

	for {
		req := <-m.req
		switch req.action {
		case actionBidReceived:
			if _, ok := m.biddingNodes[req.sourceNodeID]; !ok {
				m.biddingNodes[req.sourceNodeID] = struct{}{}

				// we have received enough bids to start the selection process.
				if len(m.biddingNodes) >= m.shard.Job.Deal.MinBids {
					return selectingBidsState
				}
			} else {
				log.Ctx(ctx).Warn().Msgf("%s ignoring duplicate bid from %s", m, req.sourceNodeID)
			}
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// Shard is selecting from the enqueued bids, and notify the selected nodes to start the computation.
func selectingBidsState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardSelectingBids)

	// randomize the candidateBids slice before returning it
	candidateBids := maps.Keys(m.biddingNodes)
	rand.Shuffle(len(candidateBids), func(i, j int) {
		candidateBids[i], candidateBids[j] = candidateBids[j], candidateBids[i]
	})

	// to hold the bids that were selected and successfully notified.
	acceptedBids := make(map[string]struct{})

	for _, candidate := range candidateBids {
		if len(acceptedBids) < m.shard.Job.Deal.Concurrency {
			err := m.node.notifyBidDecision(ctx, m.shard, candidate, true)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msgf("%s failed to notify bid acceptance to %s", m, candidate)
				continue
			} else {
				acceptedBids[candidate] = struct{}{}
			}
		} else {
			err := m.node.notifyBidDecision(ctx, m.shard, candidate, false)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf("%s failed to notify bid rejection to %s", m, candidate)
			}
		}
	}

	// updated biddingNodes to hold the accepted bids only.
	m.biddingNodes = acceptedBids

	if len(m.biddingNodes) < m.shard.Job.Deal.Concurrency {
		// we still need more bids to reach the concurrency level.
		return acceptingBidsState
	} else {
		return waitingForResultsState
	}
}

// Shard is accepting more bids to reach the concurrency level.
func acceptingBidsState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardAcceptingBids)

	for {
		req := <-m.req
		switch req.action {
		case actionBidReceived:
			if _, ok := m.biddingNodes[req.sourceNodeID]; !ok {
				err := m.node.notifyBidDecision(ctx, m.shard, req.sourceNodeID, true)
				if err != nil {
					log.Ctx(ctx).Error().Msgf("%s failed to notify bid acceptance. Will wait for more bids: %s", m, err)
				} else {
					// add the bid to the list of accepted bids.
					m.biddingNodes[req.sourceNodeID] = struct{}{}

					if len(m.biddingNodes) >= m.shard.Job.Deal.Concurrency {
						return waitingForResultsState
					}
				}
			} else {
				log.Ctx(ctx).Warn().Msgf("%s ignoring duplicate bid from %s", m, req.sourceNodeID)
			}
		case actionResultReceived:
			if _, ok := m.biddingNodes[req.sourceNodeID]; ok {
				m.completedNodes[req.sourceNodeID] = struct{}{}
			} else {
				log.Ctx(ctx).Warn().Msgf("%s ignoring result from %s", m, req.sourceNodeID)
			}
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// Shard is waiting for the results from the selected nodes, and reject any more incoming bids.
func waitingForResultsState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardWaitingForResults)

	for {
		req := <-m.req
		switch req.action {
		case actionBidReceived:
			// reject all bids at this state
			err := m.node.notifyBidDecision(ctx, m.shard, req.sourceNodeID, false)
			if err != nil {
				log.Ctx(ctx).Warn().Msgf("%s failed to notify bid rejection: %s", m, err)
			}
		case actionResultReceived:
			if _, ok := m.biddingNodes[req.sourceNodeID]; ok {
				m.completedNodes[req.sourceNodeID] = struct{}{}

				// TODO: technically we can start verifying if we have enough results compared to deal's confidence
				//  and concurrency. Though we will have ot handle the case where verification fails, but can still
				//  succeed if we wait for more results.
				if len(m.completedNodes) >= m.shard.Job.Deal.Concurrency {
					return verifyingResultsState
				}
			} else {
				log.Ctx(ctx).Warn().Msgf("%s ignoring result from %s", m, req.sourceNodeID)
			}
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// All results were received, and we are verifying them.
func verifyingResultsState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardVerifyingResults)

	verifiedResults, err := m.node.verifyShard(ctx, m.shard)
	if err != nil {
		m.errorMsg = fmt.Sprintf("failed to verify job: %s", err)
		return errorState
	}

	if len(verifiedResults) > 0 {
		return waitingToPublishResultsState
	}

	return completedState
}

func waitingToPublishResultsState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardWaitingToPublishResults)

	for {
		req := <-m.req
		switch req.action {
		case actionResultsPublished:
			// TODO: #831 verify that the published results are the same as the ones we expect, or let the verifier
			//  publish the result and not all the compute nodes.
			return completedState
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

func errorState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardError)
	errMessage := fmt.Sprintf("%s error completing job due to %s", m, m.errorMsg)
	log.Ctx(ctx).Error().Msgf(errMessage)

	ctx, span := system.GetTracer().Start(ctx, "pkg/requesterNode/ShardFSM.errorState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	err := m.node.notifyShardError(
		ctx,
		m.shard,
		errMessage,
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("%s failed to report error of job due to %s",
			m, err.Error())
	}

	return completedState
}

// we always reach this state, whether the job completed successfully or due to a failure.
func completedState(ctx context.Context, m *shardStateMachine) stateFn {
	m.transitionedTo(ctx, shardCompleted)
	return nil
}
