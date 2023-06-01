//go:build unit || !integration

package selection

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/requester/ranking"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

var (
	test1 peer.ID = peer.ID("test1")
	test2 peer.ID = peer.ID("test2")
)

type nodeSelectorTestCase struct {
	name string

	nodes  []peer.ID
	ranks  []int
	states map[peer.ID][]model.ExecutionStateType

	checkError    func(require.TestingT, error, ...any)
	expectedNodes []peer.ID
}

var testCases = []nodeSelectorTestCase{
	{
		name:       "zero",
		checkError: require.NoError,
	},
	{
		name:  "selects all",
		nodes: []peer.ID{test1, test2},
		ranks: []int{1, 1},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {model.ExecutionStateFailed},
			test2: {model.ExecutionStateFailed},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{test1, test2},
	},
	{
		name:  "honours ranking",
		nodes: []peer.ID{test1, test2},
		ranks: []int{-1, 1},
		states: map[peer.ID][]model.ExecutionStateType{
			test2: {model.ExecutionStateFailed},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{test2},
	},
}

var retryTestCases = []nodeSelectorTestCase{
	{
		name:  "does not retry successful job",
		nodes: []peer.ID{test1},
		ranks: []int{1},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {model.ExecutionStateCompleted},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{},
	},
	{
		name:  "does not retry active job",
		nodes: []peer.ID{test1},
		ranks: []int{1},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {model.ExecutionStateBidAccepted},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{},
	},
	{
		name:  "does not retry successful job after retry",
		nodes: []peer.ID{test1},
		ranks: []int{1},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {model.ExecutionStateFailed, model.ExecutionStateCompleted},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{},
	},
	{
		name:  "does not retry job after many failed retries",
		nodes: []peer.ID{test1},
		ranks: []int{1},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {
				model.ExecutionStateFailed,
				model.ExecutionStateFailed,
				model.ExecutionStateFailed,
			},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{},
	},
	{
		name:  "does not retry job on a different node",
		nodes: []peer.ID{test1, test2},
		ranks: []int{1, 2},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {
				model.ExecutionStateFailed,
				model.ExecutionStateFailed,
				model.ExecutionStateFailed,
			},
			test2: {
				model.ExecutionStateCompleted,
			},
		},
		checkError:    require.NoError,
		expectedNodes: []peer.ID{},
	},
	{
		name:  "throws error if node disappeared",
		nodes: []peer.ID{},
		ranks: []int{},
		states: map[peer.ID][]model.ExecutionStateType{
			test1: {
				model.ExecutionStateFailed,
			},
		},
		checkError:    require.Error,
		expectedNodes: []peer.ID{},
	},
}

func TestSelectNodes(t *testing.T) {
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nodeInfos := lo.Map(testCase.nodes, func(id peer.ID, _ int) model.NodeInfo {
				return model.NodeInfo{PeerInfo: peer.AddrInfo{ID: id}}
			})

			selector := NewAllNodeSelector(AllNodeSelectorParams{
				NodeDiscoverer: discovery.NewFixedDiscoverer(nodeInfos...),
				NodeRanker:     ranking.NewFixedRanker(testCase.ranks...),
			})

			nodes, err := selector.SelectNodes(context.Background(), model.NewJob())
			testCase.checkError(t, err)
			require.Len(t, nodes, len(testCase.expectedNodes))

			nodeIDs := lo.Map(nodes, func(info model.NodeInfo, _ int) peer.ID {
				return info.PeerInfo.ID
			})
			for _, node := range testCase.nodes {
				if slices.Contains(testCase.expectedNodes, node) {
					require.Contains(t, nodeIDs, node)
				} else {
					require.NotContains(t, nodeIDs, node)
				}
			}
		})
	}
}

func TestSelectNodesForRetry(t *testing.T) {
	retryTestCases := append(retryTestCases, testCases...)
	for _, testCase := range retryTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			nodeInfos := lo.Map(testCase.nodes, func(id peer.ID, _ int) model.NodeInfo {
				return model.NodeInfo{PeerInfo: peer.AddrInfo{ID: id}}
			})

			selector := NewAllNodeSelector(AllNodeSelectorParams{
				NodeDiscoverer: discovery.NewFixedDiscoverer(nodeInfos...),
				NodeRanker:     ranking.NewFixedRanker(testCase.ranks...),
			})

			job := model.NewJob()
			executions := lo.Flatten(lo.MapToSlice(testCase.states, func(key peer.ID, value []model.ExecutionStateType) []model.ExecutionState {
				return lo.Map(value, func(state model.ExecutionStateType, _ int) model.ExecutionState {
					return model.ExecutionState{JobID: job.ID(), NodeID: string(key), State: state}
				})
			}))

			nodes, err := selector.SelectNodesForRetry(context.Background(), model.NewJob(), &model.JobState{
				JobID:      job.ID(),
				Executions: executions,
			})
			testCase.checkError(t, err)
			require.Len(t, nodes, len(testCase.expectedNodes))

			nodeIDs := lo.Map(nodes, func(info model.NodeInfo, _ int) peer.ID {
				return info.PeerInfo.ID
			})
			for _, node := range testCase.nodes {
				if slices.Contains(testCase.expectedNodes, node) {
					require.Contains(t, nodeIDs, node)
				} else {
					require.NotContains(t, nodeIDs, node)
				}
			}
		})
	}
}
