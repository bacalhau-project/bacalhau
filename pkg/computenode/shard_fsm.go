package computenode

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	sync "github.com/lukemarsden/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
)

// types of actions that can be performed on a shard state machine
type shardStateAction int

const (
	// do bid on a shard
	actionBid shardStateAction = iota // must be first

	// bid was rejected, and do cancel the bid
	actionRejected

	// job has failed for some reason outside of the fsm
	actionFail

	// bid was accepted, resources are available, and do run the job
	actionRun

	// results were verified, and do publish them
	actionPublish
)

func (a shardStateAction) String() string {
	return [...]string{"ActionBid", "ActionRejected", "ActionFail", "ActionRun"}[a]
}

// request to change the state of the fsm
type shardStateRequest struct {
	action        shardStateAction
	failureReason string
}

// types of shard state machines
type shardStateType int

const (
	shardInitialState shardStateType = iota // must be first

	// Selected as a candidate shard that can be executed by this node,
	// but waiting for available capacity to be reserved before actually
	// bidding on the job.
	shardEnqueued

	// Bid on the job, and wait for the bid to be accepted.
	shardBidding

	// The bid has been accepted, and the job is now running.
	shardRunning

	// The execution of the job has completed, and publishing the results to the verifier.
	shardPublishingToVerifier

	// Waiting for the verifier to verify the results
	shardVerifyingResults

	// The results of the job has been verified, and publishing the results to the requester.
	shardPublishingToRequester

	// The job has failed due to an error.
	shardError

	// The job has been completed, either successfully, or due to an error.
	shardCompleted
)

func (s shardStateType) String() string {
	return [...]string{
		"InitialState", "Enqueued", "Bidding", "Running", "PublishingToVerifier",
		"VerifyingResults", "PublishingToRequester", "Error", "Completed"}[s]
}

type shardStateMachineManager struct {
	// map fo the shard flatID and shard state machine.
	// Used to find the shard state machine for a given flatID.
	shardStates map[string]*shardStateMachine

	// list of all shard state machines ordered by their creation time
	// according the priority defined by the capacity manager
	shardStatesList []*shardStateMachine

	mu sync.Mutex
}

func NewShardComputeStateMachineManager() (*shardStateMachineManager, error) {
	stateManager := &shardStateMachineManager{
		shardStates:     make(map[string]*shardStateMachine),
		shardStatesList: []*shardStateMachine{},
	}

	stateManager.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ComputeNode.ShardStateMachineManagerMu",
	})

	return stateManager, nil
}

// Start a new shard state machine, if it does not already exist,
// and run the fsm in a separate goroutine.
func (m *shardStateMachineManager) StartShardStateIfNecessery(
	jobID string, shardIndex int, node *ComputeNode, requirements capacitymanager.ResourceUsageData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	flatID := capacitymanager.FlattenShardID(jobID, shardIndex)

	if _, ok := m.shardStates[flatID]; !ok {
		shardState := m.newStateMachine(jobID, shardIndex, node, requirements)
		go func() {
			shardState.Run()
		}()
		m.shardStates[flatID] = shardState
		m.shardStatesList = append(m.shardStatesList, shardState)
	} // else, fsm was already running
}

// Implements CapacityTracker interface to apply the handler on enqueued shards.
func (m *shardStateMachineManager) BacklogIterator(handler func(item capacitymanager.CapacityManagerItem)) {
	for _, item := range m.GetEnqueued() {
		handler(item.capacity)
	}
}

// Implements CapacityTracker interface to apply the handler on active shards.
func (m *shardStateMachineManager) ActiveIterator(handler func(item capacitymanager.CapacityManagerItem)) {
	for _, item := range m.GetActive() {
		handler(item.capacity)
	}
}

func (m *shardStateMachineManager) GetEnqueued() []*shardStateMachine {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupCompleted()
	enqueud := []*shardStateMachine{}
	for _, i := range m.shardStatesList {
		if i.currentState == shardEnqueued {
			enqueud = append(enqueud, i)
		}
	}
	return enqueud
}

func (m *shardStateMachineManager) GetActive() []*shardStateMachine {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupCompleted()
	active := []*shardStateMachine{}
	for _, i := range m.shardStatesList {
		if i.currentState == shardBidding || i.currentState == shardRunning {
			active = append(active, i)
		}
	}
	return active
}

func (m *shardStateMachineManager) Get(flatID string) (*shardStateMachine, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fsm, ok := m.shardStates[flatID]
	return fsm, ok
}

// Since we want to keep the list of shard state machines ordered by their creation time,
// and since shards can complete at any time, we need to remove completed shards
// from the list without impacting the order of the remaining shards, and without
// having to copy things around.
// This method only removes completed shards from the beginning of the list, and is
// called inside GetEnqueued and GetActive.
func (m *shardStateMachineManager) cleanupCompleted() {
	firstActive := len(m.shardStatesList)
	for index, item := range m.shardStatesList {
		if item.currentState != shardCompleted {
			firstActive = index
			break
		}
		delete(m.shardStates, item.flatID)
	}
	m.shardStatesList = m.shardStatesList[firstActive:]
}

type shardStateMachine struct {
	jobID      string
	shardIndex int
	flatID     string
	capacity   capacitymanager.CapacityManagerItem

	manager *shardStateMachineManager
	node    *ComputeNode
	mu      sync.Mutex
	req     chan shardStateRequest

	currentState   shardStateType
	previousState  shardStateType
	resultProposal []byte
	bidSent        bool
	errorMsg       string
}

func (m *shardStateMachineManager) newStateMachine(
	jobID string, shardIndex int, node *ComputeNode, requirements capacitymanager.ResourceUsageData) *shardStateMachine {
	flatID := capacitymanager.FlattenShardID(jobID, shardIndex)

	stateMachine := &shardStateMachine{
		jobID:        jobID,
		shardIndex:   shardIndex,
		flatID:       flatID,
		manager:      m,
		node:         node,
		capacity:     capacitymanager.CapacityManagerItem{ID: flatID, Requirements: requirements},
		req:          make(chan shardStateRequest),
		currentState: shardInitialState,
	}

	stateMachine.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ComputeNode.ShardStateMachinerMu",
	})

	return stateMachine
}

func (m *shardStateMachine) String() string {
	return fmt.Sprintf("[%s] shard: %s at state: %s", m.node.id[:8], m.flatID, m.currentState)
}

// run the state machineuntil it is completed.
func (m *shardStateMachine) Run() {
	for state := enqueuedState; state != nil; {
		state = state(m)
	}
	// close the request channel.
	// Check `sendRequest` comments for more details.
	close(m.req)
}

func (m *shardStateMachine) Bid() {
	m.sendRequest(shardStateRequest{action: actionBid})
}

func (m *shardStateMachine) BidRejected() {
	m.sendRequest(shardStateRequest{action: actionRejected})
}

func (m *shardStateMachine) Execute() {
	m.sendRequest(shardStateRequest{action: actionRun})
}

func (m *shardStateMachine) Publish() {
	m.sendRequest(shardStateRequest{action: actionPublish})
}

func (m *shardStateMachine) Fail(reason string) {
	m.sendRequest(shardStateRequest{action: actionFail, failureReason: reason})
}

// send a request to the state machine by enquing it in the request channel.
// it is possible due to race condition or duplicate network events that a
// request is sent after the fsm is completed and no longer a goroutin is
// consuming from the channel, which will lead to a deadlock in the
// computenode when trying to send the request.
// To mitigagte this, we close the channel when the fsm is completed, and handle
// the panic gracefully here.
func (m *shardStateMachine) sendRequest(request shardStateRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn().Msgf("%s ignoring action after channel closed: %s", m, request.action)
		}
	}()
	m.req <- request
}

type StateFn func(*shardStateMachine) StateFn

func (m *shardStateMachine) transitionedTo(newState shardStateType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.Debug().Msgf("%s transitioning from %s -> %s", m, m.currentState, newState)
	m.previousState = m.currentState
	m.currentState = newState
}

// The job has been selected by the computeNode, and currently enqueued and waiting for
// available capacity to be reserved before actually bidding on the job.
func enqueuedState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardEnqueued)

	// trigger the bidding loop as soon as the shard state is updated to enqueued.
	go m.node.controlLoopBidOnJobs("job enqueud")

	for {
		req := <-m.req
		switch req.action {
		case actionBid:
			err := m.node.BidOnJob(context.Background(), m.jobID, m.shardIndex)
			if err != nil {
				m.errorMsg = err.Error()
				return errorState
			}

			// we've sent a bid, which means we are to send an error if anything fails afterwards
			// to let the requesterNode know fast to cancel the job or retry on another node.
			m.bidSent = true

			return biddingState
		case actionFail:
			m.errorMsg = req.failureReason
			return errorState
		default:
			log.Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the computeNode has sent a bid and is waiting for the bid to be accepted or rejected.
func biddingState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardBidding)

	for {
		req := <-m.req
		switch req.action {
		case actionRun:
			return runningState
		case actionRejected:
			return completedState
		case actionFail:
			m.errorMsg = req.failureReason
			return errorState
		default:
			log.Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the bid has been accepted and now we trigger the execution of the job.
func runningState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardRunning)

	// we get a "proposal" from this method which is not the results
	// but what the compute node verifier wants to pass to the requester
	// node verifier
	ctx := context.Background()
	proposal, err := m.node.RunShard(ctx, m.jobID, m.shardIndex)
	if err == nil {
		m.resultProposal = proposal
		return publishingToVerifierState
	} else {
		m.errorMsg = err.Error()
		return errorState
	}
}

// the job has been executed and now we verify the results.
func publishingToVerifierState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardPublishingToVerifier)

	ctx := context.Background()
	err := m.node.controller.ShardExecutionFinished(
		ctx,
		m.jobID,
		m.shardIndex,
		fmt.Sprintf("Got results proposal of length: %d", len(m.resultProposal)),
		m.resultProposal,
	)

	if err != nil {
		m.errorMsg = err.Error()
		return errorState
	} else {
		return verifyingResultsState
	}
}

// the job has been executed and the results are being published.
func verifyingResultsState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardVerifyingResults)

	for {
		req := <-m.req
		switch req.action {
		case actionPublish:
			return publishingToRequesterState
		case actionFail:
			m.errorMsg = req.failureReason
			return errorState
		default:
			log.Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the job has been executed and the results are being published.
func publishingToRequesterState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardPublishingToRequester)

	ctx := context.Background()
	err := m.node.PublishShard(ctx, m.jobID, m.shardIndex)
	if err != nil {
		m.errorMsg = err.Error()
		return errorState
	} else {
		return completedState
	}
}

func errorState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardError)
	errMessage := fmt.Sprintf("%s error completing job due to %s", m, m.errorMsg)
	log.Error().Msgf(errMessage)

	if m.bidSent {
		// we sent a bid, so we need to publish our failure to the network
		ctx := context.Background()
		err := m.node.controller.ShardError(
			ctx,
			m.jobID,
			m.shardIndex,
			errMessage,
		)
		if err != nil {
			log.Error().Msgf("%s failed to report error of job due to %s",
				m, err.Error())
		}
	}

	return completedState
}

// we always reach this state, whether the job completed successfully or due to a failure.
func completedState(m *shardStateMachine) StateFn {
	m.transitionedTo(shardCompleted)

	// once we've finished this shard - let's see if we should
	// bid on another shard or if we've finished the job
	go m.node.controlLoopBidOnJobs("job completed")
	return nil
}
