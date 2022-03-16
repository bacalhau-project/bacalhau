package system

import (
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"
)

type ResultsList struct {
	Node   string
	Cid    string
	Folder string
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

func GetJobResults(host string, port int, jobId string) (*[]ResultsList, error) {

	job, err := GetJobData(host, port, jobId)

	if err != nil {
		return nil, err
	}

	return ProcessJobIntoResults(job)
}

func ProcessJobIntoResults(job *types.Job) (*[]ResultsList, error) {
	results := []ResultsList{}

	log.Debug().Msgf("All job states: %+v", job)

	log.Debug().Msgf("Number of job states created: %d", len(job.State))

	for node := range job.State {

		cid := ""

		if len(job.State[node].Outputs) > 0 {
			cid = job.State[node].Outputs[0].Cid
		}

		results = append(results, ResultsList{
			Node:   node,
			Cid:    cid,
			Folder: GetResultsDirectory(job.Id, node),
		})
	}

	log.Debug().Msgf("Number of results created: %d", len(results))

	return &results, nil
}

func FetchJobResult(results ResultsList) error {
	resultsFolder, err := GetSystemDirectory(results.Folder)
	if err != nil {
		return err
	}
	if _, err := os.Stat(resultsFolder); !os.IsNotExist(err) {
		return nil
	}
	log.Debug().Msgf("Fetching results for job %s ---> %s\n", results.Cid, results.Folder)
	resultsFolder, err = EnsureSystemDirectory(results.Folder)
	if err != nil {
		return fmt.Errorf("Error ensuring system directory: %s", err)
	}
	output, err := RunCommandGetResults("ipfs", []string{
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
