package computenode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	sync "github.com/lukemarsden/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
)

// types of actions that can be performed on a shard state machine
type shardStateAction int

const (
	// do bid on a shard
	actionBid shardStateAction = iota // must be first

	// bid was rejected, and do cancel the bid
	actionBidRejected

	// cancel the job mainly due to requester node already accepted other bids.
	// Can only cancel a shard before a bid is sent. After that the action will be ignored and you can
	// only fail the shard.
	actionCancel

	// job has failed for some reason outside of the fsm
	actionFail

	// bid was accepted, resources are available, and do run the job
	actionRun

	// proposed results were rejected
	actionResultsRejected

	// results were verified, and do publish them
	actionPublish
)

func (a shardStateAction) String() string {
	return [...]string{
		"ActionBid", "ActionBidRejected", "ActionCancel", "ActionFail",
		"ActionRun", "ActionResultsRejected", "ActionPublish"}[a]
}

// request to change the state of the fsm
type shardStateRequest struct {
	action shardStateAction
	reason string
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

	// The job has been canceled, mainly due to other bids already accepted.
	shardCancelled

	// The job has failed due to an error.
	shardError

	// The job has been completed, either successfully, or due to an error.
	shardCompleted
)

func (s shardStateType) String() string {
	return [...]string{
		"InitialState", "Enqueued", "Bidding", "Running", "PublishingToVerifier",
		"VerifyingResults", "PublishingToRequester", "Canceled", "Error", "Completed"}[s]
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
func (m *shardStateMachineManager) StartShardStateIfNecessary(
	ctx context.Context, shard model.JobShard, n *ComputeNode, requirements model.ResourceUsageData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.shardStates[shard.ID()]; !ok {
		shardState := m.newStateMachine(shard, n, requirements)

		// ANCHOR: Start of the shard state machine span
		ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardStateMachineManager.StartShardStateIfNecessary") //nolint:govet
		defer span.End()
		ctx = system.AddNodeIDToBaggage(ctx, n.ID)
		system.AddNodeIDFromBaggageToSpan(ctx, span)

		go func() {
			shardState.Run(ctx)
		}()
		m.shardStates[shard.ID()] = shardState
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
		delete(m.shardStates, item.Shard.ID())
	}
	m.shardStatesList = m.shardStatesList[firstActive:]
}

type shardStateMachine struct {
	Shard    model.JobShard
	capacity capacitymanager.CapacityManagerItem

	manager *shardStateMachineManager
	node    *ComputeNode
	req     chan shardStateRequest

	currentState  shardStateType
	previousState shardStateType

	runOutput       *model.RunCommandResult
	resultProposal  []byte
	cancellationMsg string
	errorMsg        string

	notifyOnFailure bool
}

func (m *shardStateMachineManager) newStateMachine(
	shard model.JobShard, node *ComputeNode, requirements model.ResourceUsageData) *shardStateMachine {
	stateMachine := &shardStateMachine{
		Shard:        shard,
		manager:      m,
		node:         node,
		capacity:     capacitymanager.CapacityManagerItem{Shard: shard, Requirements: requirements},
		req:          make(chan shardStateRequest),
		currentState: shardInitialState,
	}

	return stateMachine
}

func (m *shardStateMachine) String() string {
	return fmt.Sprintf("[%s] shard: %s at state: %s", m.node.ID[:8], m.Shard, m.currentState)
}

// run the state machineuntil it is completed.
func (m *shardStateMachine) Run(ctx context.Context) {
	for state := enqueuedState; state != nil; {
		// TODO: #559 Should we create a new context and span for each state execution?
		state = state(ctx, m)
	}
	// close the request channel.
	// Check `sendRequest` comments for more details.
	close(m.req)
}

func (m *shardStateMachine) Bid(ctx context.Context) {
	m.sendRequest(ctx, shardStateRequest{action: actionBid})
}

func (m *shardStateMachine) BidRejected(ctx context.Context) {
	m.sendRequest(ctx, shardStateRequest{action: actionBidRejected})
}

func (m *shardStateMachine) Execute(ctx context.Context) {
	m.sendRequest(ctx, shardStateRequest{action: actionRun})
}

func (m *shardStateMachine) ResultsRejected(ctx context.Context) {
	m.sendRequest(ctx, shardStateRequest{action: actionResultsRejected})
}

func (m *shardStateMachine) Publish(ctx context.Context) {
	m.sendRequest(ctx, shardStateRequest{action: actionPublish})
}

// Can only cancel a shard before a bid is sent. After that the action will be ignored and you can
// only fail the shard.
func (m *shardStateMachine) Cancel(ctx context.Context, reason string) {
	m.sendRequest(ctx, shardStateRequest{action: actionCancel, reason: reason})
}

func (m *shardStateMachine) Fail(ctx context.Context, reason string) {
	m.sendRequest(ctx, shardStateRequest{action: actionFail, reason: reason})
}

// send a request to the state machine by enquing it in the request channel.
// it is possible due to race condition or duplicate network events that a
// request is sent after the fsm is completed and no longer a goroutin is
// consuming from the channel, which will lead to a deadlock in the
// computenode when trying to send the request.
// To mitigagte this, we close the channel when the fsm is completed, and handle
// the panic gracefully here.
func (m *shardStateMachine) sendRequest(ctx context.Context, request shardStateRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Ctx(ctx).Warn().Msgf("%s ignoring action after channel closed: %s", m, request.action)
		}
	}()
	m.req <- request
}

type StateFn func(context.Context, *shardStateMachine) StateFn

func (m *shardStateMachine) transitionedTo(ctx context.Context, newState shardStateType, reasons ...string) {
	reason := ""
	if reasons != nil {
		reason = " due to " + strings.Join(reasons, ", ")
	}
	log.Ctx(ctx).Debug().Msgf("%s transitioning from %s -> %s%s", m, m.currentState, newState, reason)
	m.previousState = m.currentState
	m.currentState = newState
}

// ------------------------------------
// Job Shard State Machine Functions
// ------------------------------------

// The job has been selected by the computeNode, and currently enqueued and waiting for
// available capacity to be reserved before actually bidding on the job.
func enqueuedState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardEnqueued)

	for {
		req := <-m.req
		switch req.action {
		case actionBid:
			err := m.node.notifyBidJob(ctx, m.Shard)
			if err != nil {
				m.errorMsg = err.Error()
				return errorState
			}

			// we've sent a bid, which means we are to send an error if anything fails afterwards
			// to let the requesterNode know fast to cancel the job or retry on another node.
			m.notifyOnFailure = true

			return biddingState
		case actionCancel:
			m.cancellationMsg = req.reason
			return cancelledState
		case actionFail:
			m.errorMsg = req.reason
			return errorState
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the computeNode has sent a bid and is waiting for the bid to be accepted or rejected.
func biddingState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardBidding)

	for {
		req := <-m.req
		switch req.action {
		case actionRun:
			return runningState
		case actionBidRejected:
			return completedState
		case actionFail:
			m.errorMsg = req.reason
			return errorState
		default:
			// TODO: #832 Get a lot of these, should we care? 'Bidding ignoring unknown action: ActionBid [NodeID:QmUhzQME]'
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the bid has been accepted and now we trigger the execution of the job.
func runningState(ctx context.Context, m *shardStateMachine) StateFn {
	// TODO: #558 Should we create a new span every time there's a state transition?
	m.transitionedTo(ctx, shardRunning)

	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardFSM.runningState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.Shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	// we get a "proposal" from this method which is not the results
	// but what the compute node verifier wants to pass to the requester
	// node verifier
	proposal, runOutput, err := m.node.RunShard(ctx, m.Shard)
	m.runOutput = runOutput
	if err == nil {
		m.resultProposal = proposal
		return publishingToVerifierState
	} else {
		m.errorMsg = err.Error()
		return errorState
	}
}

// the job has been executed and now we verify the results.
func publishingToVerifierState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardPublishingToVerifier)

	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardFSM.publishingToVerifierState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.Shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	err := m.node.notifyShardExecutionFinished(
		ctx,
		m.Shard,
		fmt.Sprintf("Got results proposal of length: %d", len(m.resultProposal)),
		m.resultProposal,
		m.runOutput,
	)

	if err != nil {
		m.errorMsg = err.Error()
		return errorState
	} else {
		return verifyingResultsState
	}
}

// the job has been executed and the results are being published.
func verifyingResultsState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardVerifyingResults)

	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardFSM.verifyingResultsState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.Shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	for {
		req := <-m.req
		switch req.action {
		case actionPublish:
			return publishingToRequesterState
		case actionResultsRejected:
			// no need to publish an event since the requester node
			// already published a failure event
			m.notifyOnFailure = false
			return completedState
		case actionFail:
			m.errorMsg = req.reason
			return errorState
		default:
			log.Ctx(ctx).Warn().Msgf("%s ignoring unknown action: %s", m, req.action)
		}
	}
}

// the job has been executed and the results are being published.
func publishingToRequesterState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardPublishingToRequester)

	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardFSM.publishingToRequesterState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.Shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	err := m.node.PublishShard(ctx, m.Shard)
	if err != nil {
		m.errorMsg = err.Error()
		return errorState
	} else {
		return completedState
	}
}

func errorState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardError)

	//nolint:lll
	// TODO: #833 We throw an error into our logs for every user error, we should split things into User Errors and System Errors. If they have a bad binary, that's their fault, not ours.
	errMessage := fmt.Sprintf("errorState: error completing job due to: %s", m.errorMsg)
	log.Ctx(ctx).Error().Msgf(errMessage)

	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode/ShardFSM.errorState")
	defer span.End()
	ctx = system.AddJobIDToBaggage(ctx, m.Shard.Job.ID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	if m.notifyOnFailure {
		// we sent a bid, so we need to publish our failure to the network
		err := m.node.notifyShardError(
			ctx,
			m.Shard,
			errMessage,
			m.runOutput,
		)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("errorState: failed to report error of job due to %s", err.Error())
		}
	}

	return completedState
}

func cancelledState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardCancelled, m.cancellationMsg)
	// no notifications need to be sent here as you can only cancel a shard before a bid is sent.
	return completedState
}

// we always reach this state, whether the job completed successfully or due to a failure.
func completedState(ctx context.Context, m *shardStateMachine) StateFn {
	m.transitionedTo(ctx, shardCompleted)
	return nil
}
