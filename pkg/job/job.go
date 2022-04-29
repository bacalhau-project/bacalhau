package job

import (
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/jsonrpc"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

func GetJobData(host string, port int, jobId string) (*types.Job, error) {
	args := &types.ListArgs{}
	result := &types.ListResponse{}
	err := jsonrpc.JsonRpcMethod(host, port, "List", args, result)
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

func GetJobResults(host string, port int, jobId string) (*[]types.ResultsList, error) {

	job, err := GetJobData(host, port, jobId)

	if err != nil {
		return nil, err
	}

	return ProcessJobIntoResults(job)
}

func ProcessJobIntoResults(job *types.Job) (*[]types.ResultsList, error) {
	results := []types.ResultsList{}

	log.Debug().Msgf("All job states: %+v", job)

	log.Debug().Msgf("Number of job states created: %d", len(job.State))

	for node := range job.State {

		cid := ""

		if len(job.State[node].Outputs) > 0 {
			cid = job.State[node].Outputs[0].Cid
		}

		results = append(results, types.ResultsList{
			Node:   node,
			Cid:    cid,
			Folder: system.GetResultsDirectory(job.Id, node),
		})
	}

	log.Debug().Msgf("Number of results created: %d", len(results))

	return &results, nil
}

func FetchJobResult(results types.ResultsList) error {
	resultsFolder, err := system.GetSystemDirectory(results.Folder)
	if err != nil {
		return err
	}
	if _, err := os.Stat(resultsFolder); !os.IsNotExist(err) {
		return nil
	}
	log.Debug().Msgf("Fetching results for job %s ---> %s\n", results.Cid, results.Folder)
	resultsFolder, err = system.EnsureSystemDirectory(results.Folder)
	if err != nil {
		return fmt.Errorf("Error ensuring system directory: %s", err)
	}
	output, err := system.RunCommandGetResults("ipfs", []string{
		"get",
		results.Cid,
		"--output",
		resultsFolder,
	})
	if err != nil {
		return fmt.Errorf(`Error getting fetching results:
Output: %s
Error: %s`, output, err)
	}
	return nil
}

func FetchJobResults(host string, port int, jobId string) error {
	data, err := GetJobResults(host, port, jobId)
	if err != nil {
		return err
	}

	for _, row := range *data {
		err = FetchJobResult(row)
		if err != nil {
			return err
		}
	}

	return nil
}

func ListJobs(
	rpcHost string,
	rpcPort int,
) (*types.ListResponse, error) {
	args := &types.ListArgs{}
	result := &types.ListResponse{}
	err := jsonrpc.JsonRpcMethod(rpcHost, rpcPort, "List", args, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func RunJob(
	cids []string,
	env []string,
	image, entrypoint string,
	concurrency int,
	rpcHost string,
	rpcPort int,
	skipSyntaxChecking bool,
) (*types.Job, error) {

	if concurrency <= 0 {
		return nil, fmt.Errorf("Concurrency must be >= 1")
	}

	if !skipSyntaxChecking {
		err := system.CheckBashSyntax([]string{entrypoint})
		if err != nil {
			return nil, err
		}
	}

	jobInputs := []types.StorageSpec{}

	for _, cid := range cids {
		jobInputs = append(jobInputs, types.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Cid:    cid,
		})
	}

	spec := &types.JobSpec{
		Image:      image,
		Entrypoint: entrypoint,
		Env:        env,
		Inputs:     jobInputs,
	}

	deal := &types.JobDeal{
		Concurrency: concurrency,
	}

	args := &types.SubmitArgs{
		Spec: spec,
		Deal: deal,
	}

	result := &types.Job{}

	err := jsonrpc.JsonRpcMethod(rpcHost, rpcPort, "Submit", args, result)
	if err != nil {
		return nil, err
	}

	//we got our result in result
	// fmt.Printf("submit job: %+v\nreply job: %+v\n\n", args.Job, result)
	// fmt.Printf("to view all files by all nodes\n")
	// fmt.Printf("------------------------------\n\n")
	// fmt.Printf("tree ./outputs/%s\n\n", job.Id)
	// fmt.Printf("to open all metrics pngs\n")
	// fmt.Printf("------------------------\n\n")
	// fmt.Printf("find ./outputs/%s -type f -name 'metrics.png' 2> /dev/null | while read -r FILE ; do xdg-open \"$FILE\" ; done\n\n", job.Id)
	log.Info().Msgf("Submitted Job Id: %s\n", result.Id)

	return result, nil
}
