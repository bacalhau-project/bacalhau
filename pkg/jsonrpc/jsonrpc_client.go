package jsonrpc

import (
	"fmt"
	"net/rpc"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

func JsonRpcMethod(
	host string,
	port int,
	method string,
	req, res interface{},
) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JSONRpcServer.%s", method), req, res)
}

func GetJobData(host string, port int, jobId string) (*types.Job, error) {
	args := &types.ListArgs{}
	result := &types.ListResponse{}
	err := JsonRpcMethod(host, port, "List", args, result)
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

func ListJobs(
	rpcHost string,
	rpcPort int,
) (*types.ListResponse, error) {
	args := &types.ListArgs{}
	result := &types.ListResponse{}
	err := JsonRpcMethod(rpcHost, rpcPort, "List", args, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SubmitJob(
	spec *types.JobSpec,
	deal *types.JobDeal,
	rpcHost string,
	rpcPort int,
) (*types.Job, error) {

	args := &types.SubmitArgs{
		Spec: spec,
		Deal: deal,
	}

	result := &types.Job{}

	err := JsonRpcMethod(rpcHost, rpcPort, "Submit", args, result)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Submitted Job Id: %s", result.Id)

	return result, nil
}
