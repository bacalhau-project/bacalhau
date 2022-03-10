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

func init() {
	submitCmd.PersistentFlags().StringSliceVar(
		&jobCids, "cids", []string{},
		`The cids of the data used by the job (comma separated, or specify multiple times)`,
	)
	submitCmd.PersistentFlags().StringArrayVar(
		&jobCommands, "commands", []string{},
		`The commands for the job (specify multiple times for multiple commands)`,
	)
	submitCmd.PersistentFlags().IntVar(
		&jobConcurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
}

func SubmitJob(
	commands, cids []string,
	concurrency int,
	rpcHost string,
	rpcPort int,
) (*types.Job, error) {
	if len(commands) <= 0 {
		return nil, fmt.Errorf("Empty command list")
	}

	if len(cids) <= 0 {
		return nil, fmt.Errorf("Empty input list")
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
			jsonrpcHost,
			jsonrpcPort,
		)
		return err
	},
}
