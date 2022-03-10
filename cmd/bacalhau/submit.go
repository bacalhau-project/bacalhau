package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/spf13/cobra"
)

var jobCids []string
var jobCommands []string
var jobConcurrency int
var jobConfidence int
var jobTolerance float32

func init() {
	submitCmd.PersistentFlags().StringSliceVar(
		&jobCids, "cids", []string{},
		`The cids of the data used by the job (comma separated, or specify multiple times)`,
	)
	submitCmd.PersistentFlags().StringSliceVar(
		&jobCommands, "commands", []string{},
		`The commands for the job (comma separated, or specify multiple times)`,
	)
	submitCmd.PersistentFlags().IntVar(
		&jobConcurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
	submitCmd.PersistentFlags().IntVar(
		&jobConfidence, "confidence", 1,
		`How many nodes should agree on a result before we accept it`,
	)
	submitCmd.PersistentFlags().Float32Var(
		&jobTolerance, "tolerance", 0,
		`The percentage difference between two results to count them as the "same"`,
	)
}

func SubmitJob(
	commands, cids []string,
	concurrency, confidence int,
	tolerance float32,
	rpcHost string,
	rpcPort int,
) (*types.Job, error) {

	commands = []string{
		`python3 -c "import random; import time; x = '0'*1024*1024*100 if random.random() > 0 else print('noalloc'); time.sleep(10)"`,
	}

	if len(commands) <= 0 {
		return nil, fmt.Errorf("Empty command list")
	}

	// if len(cids) <= 0 {
	// 	return nil, fmt.Errorf("Empty input list")
	// }

	if concurrency <= 0 {
		return nil, fmt.Errorf("Concurrency must be >= 1")
	}

	if confidence > concurrency {
		return nil, fmt.Errorf("Confidence cannot be more than concurrency")
	}

	if confidence <= 0 {
		return nil, fmt.Errorf("Confidence must be >= 1")
	}

	if tolerance < 0 || tolerance >= 1 {
		return nil, fmt.Errorf("Tolerance must be >= 0 and < 1")
	}

	jobInputs := []types.JobStorage{}

	for _, cid := range cids {
		jobInputs = append(jobInputs, types.JobStorage{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Cid:    cid,
		})
	}

	spec := &types.JobSpec{
		Commands: commands,
		Inputs:   jobInputs,
	}

	deal := &types.JobDeal{
		Concurrency: concurrency,
		Confidence:  confidence,
		Tolerance:   tolerance,
	}

	args := &internal.SubmitArgs{
		Spec: spec,
		Deal: deal,
	}

	result := &types.Job{}

	err := JsonRpcMethodWithConnection(rpcHost, rpcPort, "Submit", args, result)
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
	fmt.Printf("job id: %s\n", result.Id)

	return result, nil
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		_, err := SubmitJob(
			jobCommands,
			jobCids,
			jobConcurrency,
			jobConfidence,
			jobTolerance,
			jsonrpcHost,
			jsonrpcPort,
		)
		return err
	},
}
