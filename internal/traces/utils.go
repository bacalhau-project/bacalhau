package traces

import (
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

// return 2 lists of job ids - correct and incorrect
func ProcessResults(job *types.Job, data *[]system.ResultsList) ([]string, []string, error) {
	// TODO: load the job so we can get at Deal.Tolerance
	clustered := TraceCollection{
		Traces:    []Trace{},
		Tolerance: job.Deal.Tolerance,
	}

	for _, row := range *data {
		resultsFolder, err := system.GetSystemDirectory(row.Folder)
		if err != nil {
			return []string{}, []string{}, err
		}

		if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
			fmt.Printf("continue not exist\n")
			continue
		}
		clustered.Traces = append(clustered.Traces, Trace{
			ResultId: row.Cid,
			Filename: resultsFolder + "/metrics.log",
		})
	}

	// TODO: actually process the results and return them
	// these are lists of CIDs of the results
	// correctGroup, incorrectGroup, err := clustered.Cluster()

	// if err != nil {
	// 	return []string{}, []string{}, err
	// }

	return []string{}, []string{}, nil
}
