package requester

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Endpoint is the frontend and entry point to the requester node for the end users to submit, update and cancel jobs.
type Endpoint interface {
	// SubmitJob submits a new job to the network.
	SubmitJob(context.Context, model.JobCreatePayload) (*model.Job, error)
	// UpdateDeal updates an existing job.
	UpdateDeal(context.Context, string, model.Deal) error
	// CancelJob cancels an existing job.
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
}

// NodeDiscoverer discovers nodes in the network that are suitable to execute a job.
type NodeDiscoverer interface {
	FindNodes(ctx context.Context, job model.Job) ([]model.NodeInfo, error)
}

// NodeRanker ranks nodes based on their suitability to execute a job.
type NodeRanker interface {
	RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]NodeRank, error)
}

type NodeInfoStore interface {
	// Add adds a node info to the repo.
	Add(ctx context.Context, nodeInfo model.NodeInfo) error
	// Get returns the node info for the given peer ID.
	Get(ctx context.Context, peerID peer.ID) (model.NodeInfo, error)
	// List returns a list of nodes
	List(ctx context.Context) ([]model.NodeInfo, error)
	// ListForEngine returns a list of nodes that support the given engine.
	ListForEngine(ctx context.Context, engine model.Engine) ([]model.NodeInfo, error)
	// Delete deletes a node info from the repo.
	Delete(ctx context.Context, peerID peer.ID) error
}

// NodeRank represents a node and its rank. The higher the rank, the more preferable a node is to execute the job.
// A negative rank means the node is not suitable to execute the job.
type NodeRank struct {
	NodeInfo model.NodeInfo
	Rank     int
}

// StartJobRequest triggers the scheduling of a job.
type StartJobRequest struct {
	Job model.Job
}

type CancelJobRequest struct {
	JobID string
}

type CancelJobResult struct {
}
