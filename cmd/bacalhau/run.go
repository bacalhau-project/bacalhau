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
var jobEnv []string
var jobImage string
var jobEntrypoint string
var jobConcurrency int
var skipSyntaxChecking bool

func init() {
	runCmd.PersistentFlags().StringSliceVar(
		&jobCids, "cids", []string{},
		`The cids of the data used by the job (comma separated, or specify multiple times)`,
	)
	runCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	runCmd.PersistentFlags().IntVar(
		&jobConcurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
	runCmd.PersistentFlags().StringVar(
		&jobImage, "image", "ubuntu:latest",
		`What image do we use for the job`,
	)
	runCmd.PersistentFlags().StringVar(
		&jobEntrypoint, "entrypoint", "",
		`The entrypoint to use for the container`,
	)
	runCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)
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

	// for testing the tracing - just run a job that allocates some memory
	if os.Getenv("BACALHAU_MOCK_JOB") != "" {
		entrypoint = `python3 -c "import time; x = '0'*1024*1024*100; time.sleep(10)"`
	}

	if concurrency <= 0 {
		return nil, fmt.Errorf("Concurrency must be >= 1")
	}

	if !skipSyntaxChecking {
		err := system.CheckBashSyntax([]string{entrypoint})
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

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint
		_, err := RunJob(
			jobCids,
			jobEnv,
			jobImage,
			jobEntrypoint,
			jobConcurrency,
			jsonrpcHost,
			jsonrpcPort,
			skipSyntaxChecking,
		)
		return err
	},
}
