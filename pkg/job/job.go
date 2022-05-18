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
		results = append(results, types.ResultsList{
			Node:   node,
			Cid:    job.State[node].ResultsId,
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

func ConstructJob(
	engine string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
) (*types.JobSpec, *types.JobDeal, error) {
	if concurrency <= 0 {
		return nil, nil, fmt.Errorf("Concurrency must be >= 1")
	}

	jobInputs := []types.StorageSpec{}
	jobOutputs := []types.StorageSpec{}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return nil, nil, fmt.Errorf("Invalid input volume: %s", inputVolume)
		}
		jobInputs = append(jobInputs, types.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Cid:    slices[0],
			Path:   slices[1],
		})
	}

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 {
			return nil, nil, fmt.Errorf("Invalid output volume: %s", outputVolume)
		}
		jobOutputs = append(jobInputs, types.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Name:   slices[0],
			Path:   slices[1],
		})
	}

	spec := &types.JobSpec{
		Engine: engine,
		Vm: types.JobSpecVm{
			Image:      image,
			Entrypoint: entrypoint,
			Env:        env,
		},

		Inputs:  jobInputs,
		Outputs: jobOutputs,
	}

	deal := &types.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
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

	err := jsonrpc.JsonRpcMethod(rpcHost, rpcPort, "Submit", args, result)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Submitted Job Id: %s", result.Id)

	return result, nil
}

func RunJob(
	engine string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
	rpcHost string,
	rpcPort int,
	skipSyntaxChecking bool,
) (*types.Job, error) {

	spec, deal, err := ConstructJob(engine, inputVolumes, outputVolumes, env, entrypoint, image, concurrency)

	if err != nil {
		return nil, err
	}

	if !skipSyntaxChecking {
		err := system.CheckBashSyntax(entrypoint)
		if err != nil {
			return nil, err
		}
	}

	return SubmitJob(spec, deal, rpcHost, rpcPort)
}
