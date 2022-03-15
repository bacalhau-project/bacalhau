package bacalhau

import (
	"strings"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

var listOutputFormat string
var tableOutputWide bool

func JsonRpcMethod(method string, req, res interface{}) error {
	return system.JsonRpcMethod(
		jsonrpcHost,
		jsonrpcPort,
		method,
		req,
		res,
	)
}

func getString(st string) string {
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
	return system.GetJobData(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}

func getJobResults(jobId string) (*[]system.ResultsList, error) {
	return system.GetJobResults(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}

func fetchJobResults(jobId string) error {
	return system.FetchJobResults(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}
