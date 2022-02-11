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

func JsonRpcMethod(method string, req, res interface{}) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", jsonrpcHost, jsonrpcPort))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JobServer.%s", method), req, res)
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

func getJobData(jobId string) (*types.JobData, error) {
	args := &internal.ListArgs{}
	result := &types.ListResponse{}
	err := JsonRpcMethod("List", args, result)
	if err != nil {
		return nil, err
	}

	var foundJob *types.Job

	for _, job := range result.Jobs {
		if strings.HasPrefix(job.Id, jobId) {
			foundJob = &job
		}
	}

	if foundJob == nil {
		return nil, fmt.Errorf("Could not find job: %s", jobId)
	}

	data := types.JobData{
		Job:     *foundJob,
		State:   result.JobState[foundJob.Id],
		Status:  result.JobStatus[foundJob.Id],
		Results: result.JobResults[foundJob.Id],
	}

	return &data, nil
}

func getJobResults(jobId string) (*[]ResultsList, error) {

	data, err := getJobData(jobId)

	if err != nil {
		return nil, err
	}

	results := []ResultsList{}
	for node := range data.State {
		results = append(results, ResultsList{
			Node:   node,
			Cid:    data.Results[node],
			Folder: system.GetResultsDirectory(data.Job.Id, node),
		})
	}

	return &results, nil
}
