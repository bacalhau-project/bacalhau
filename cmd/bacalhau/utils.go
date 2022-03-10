package bacalhau

import (
	"fmt"
	"net/rpc"
	"strings"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

var listOutputFormat string
var tableOutputWide bool

func JsonRpcMethodWithConnection(
	host string,
	port int,
	method string,
	req, res interface{},
) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JobServer.%s", method), req, res)
}

func JsonRpcMethod(method string, req, res interface{}) error {
	return JsonRpcMethodWithConnection(
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

type ResultsList struct {
	Node   string
	Cid    string
	Folder string
}

func getJobData(jobId string) (*types.Job, error) {
	args := &internal.ListArgs{}
	result := &types.ListResponse{}
	err := JsonRpcMethod("List", args, result)
	if err != nil {
		return nil, err
	}

	for _, jobData := range result.Jobs {
		if strings.HasPrefix(jobData.Id, jobId) {
			return jobData, nil
		}
	}

	return nil, fmt.Errorf("Could not find job: %s", jobId)
}

func getJobResults(jobId string) (*[]ResultsList, error) {

	job, err := getJobData(jobId)

	if err != nil {
		return nil, err
	}

	results := []ResultsList{}

	for node := range job.State {

		cid := ""

		if len(job.State[node].Outputs) > 0 {
			cid = job.State[node].Outputs[0].Cid
		}

		results = append(results, ResultsList{
			Node:   node,
			Cid:    cid,
			Folder: system.GetResultsDirectory(job.Id, node),
		})
	}

	return &results, nil
}
