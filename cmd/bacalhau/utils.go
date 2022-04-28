package bacalhau

import (
	"strings"

	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
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
	parts := strings.Split(id, "-")
	return parts[0]
}

func getJobData(jobId string) (*types.Job, error) {
	return jobutils.GetJobData(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}

func getJobResults(jobId string) (*[]types.ResultsList, error) {
	return jobutils.GetJobResults(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}

func fetchJobResults(jobId string) error {
	return jobutils.FetchJobResults(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}
