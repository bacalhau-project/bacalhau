package bacalhau

import (
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var jobCids []string
var jobCommands []string
var jobConcurrency int
var jobConfidence int
var jobTolerance float64
var skipSyntaxChecking bool

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
	submitCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)
	// this is currently fixed to the "real memory" usage of the validator (traces) module
	//
	// some example numbers for 3 jobs that are the same:
	// -0.1012000000000027
	// 0.08959999999999875
	// 0.011600000000003875
	//
	// and some example numbers for 3 jobs that are different:
	// 57.8706
	// -29.044399999999996
	// -28.826199999999993
	//
	// so - "0.5" seems to be a reasonable "gap" to count results as the same
	// TODO: have the tolerance be scaled somehow so you can give a number between 0 and 1
	// TODO: have the tolerance apply to difference validation modules (currently: psrecord + real memory usage)
	submitCmd.PersistentFlags().Float64Var(
		&jobTolerance, "tolerance", 0.5,
		`The allowable difference between two results to count them as the "same"`,
	)
}

func SubmitJob(
	commands, cids []string,
	concurrency, confidence int,
	tolerance float64,
	rpcHost string,
	rpcPort int,
	skipSyntaxChecking bool,
) (*types.Job, error) {

	// for testing the tracing - just run a job that allocates some memory
	if os.Getenv("BACALHAU_MOCK_JOB") != "" {
		commands = []string{
			`python3 -c "import time; x = '0'*1024*1024*100; time.sleep(10)"`,
		}
	}

	if len(commands) <= 0 {
		return nil, fmt.Errorf("Empty command list")
	}

	if concurrency <= 0 {
		return nil, fmt.Errorf("Concurrency must be >= 1")
	}

	if confidence > concurrency {
		return nil, fmt.Errorf("Confidence cannot be more than concurrency")
	}

	if confidence <= 0 {
		return nil, fmt.Errorf("Confidence must be >= 1")
	}

	if tolerance < 0 {
		return nil, fmt.Errorf("Tolerance must be >= 0")
	}

	if !skipSyntaxChecking {
		err := system.CheckBashSyntax(jobCommands)
		if err != nil {
			return nil, err
		}
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

	args := &types.SubmitArgs{
		Spec: spec,
		Deal: deal,
	}

	result := &types.Job{}

	err := system.JsonRpcMethod(rpcHost, rpcPort, "Submit", args, result)
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

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint
		_, err := SubmitJob(
			jobCommands,
			jobCids,
			jobConcurrency,
			jobConfidence,
			jobTolerance,
			jsonrpcHost,
			jsonrpcPort,
			skipSyntaxChecking,
		)
		return err
	},
}
