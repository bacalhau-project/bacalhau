package bacalhau

import (
	"strings"

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
	return jsonrpc.GetJobData(
		jsonrpcHost,
		jsonrpcPort,
		jobId,
	)
}
