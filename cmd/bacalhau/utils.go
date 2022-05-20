package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/jsonrpc"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

var listOutputFormat string
var tableOutputWide bool

func JsonRpcMethod(method string, req, res interface{}) error {
	return jsonrpc.JsonRpcMethod(
		jsonrpcHost,
		jsonrpcPort,
		method,
		req,
		res,
	)
}

func shortenString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < 20 {
		return st
	}

	return st[:20] + "..."
}

func shortId(id string) string {
	return id[:8]
}

func getJobData(jobId string) (*types.Job, error) {
	return jsonrpc.GetJobData(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}

func getJobResult(job *types.Job, state *types.JobState) string {
	return "/" + job.Spec.Verifier + "/" + state.ResultsId
}
