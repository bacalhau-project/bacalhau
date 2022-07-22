package bacalhau

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	pjob "github.com/filecoin-project/bacalhau/pkg/job"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const CompleteStatus = "Complete"

var jobEngine string
var jobVerifier string
var jobInputs []string
var jobInputUrls []string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobEnv []string
var jobConcurrency int
var jobIpfsGetTimeOut int
var jobCPU string
var jobMemory string
var jobGPU string
var skipSyntaxChecking bool
var waitForJobToFinishAndPrintOutput bool
var jobLabels []string

type CheckJobStatesFunction func(map[string]executor.JobStateType) (bool, error)

func GetJobStates(ctx context.Context, jobID string) (map[string]executor.JobStateType, error) {

	job, ok, err := getAPIClient().Get(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf(
			"error fetching job %s: %v", jobID, err)
	}
	if !ok {
		return nil, nil
	}

	states := map[string]executor.JobStateType{}
	for id, state := range job.State {
		states[id] = state.State
	}

	return states, nil
}

func WaitForJobWithLogs(
	ctx context.Context,
	jobID string,
	shouldLog bool,
	finalJobState executor.JobState,
	checkJobStateFunctions ...CheckJobStatesFunction,
) error {
	waiter := &system.FunctionWaiter{
		Name:        "wait for job",
		MaxAttempts: 100,
		Delay:       time.Second * 1,
		Handler: func() (bool, error) {

			// count := 0
			// if count < 1 {
			// }
			// count++

			// time.Sleep(time.Second * 15)

			// sleep till states are there
			for {
				time.Sleep(time.Second * 5)
				states, err := GetJobStates(ctx, jobID)
				if err != nil {
					fmt.Printf("error is : %v", err)
				}
				if states != nil {
					break
				}
			}
			// load the current states of the job
			states, err := GetJobStates(ctx, jobID)
			if err != nil {
				return false, err
			}
			var Status string
			for _, status := range states {
				if status.String() == "Complete" {
					Status = "Complete"
				}
			}
			if Status == "Complete" {
				return true, nil

			}

			if shouldLog {
				spew.Dump(states)
			}

			allOk := true
			fmt.Printf("Waiter %#v\n", states)
			fmt.Printf("Waiter %#v\n", checkJobStateFunctions)
			fmt.Printf("Waiter States %#v\n", states)
			for _, checkFunction := range checkJobStateFunctions {

				stepOk, err := checkFunction(states)
				if err != nil {
					return false, err
				}
				if !stepOk {
					allOk = false
				}
			}

			// If all the jobs are in terminal states, then nothing is going
			// to change if we keep polling, so we should exit early.
			allTerminal := finalJobState.State.String() != "Complete"

			for _, state := range states {
				terminate := !state.IsTerminal() || allTerminal
				fmt.Print(terminate, finalJobState.Status)
				if Status == "Complete" {
					return allOk, nil
				}
				if allTerminal {
					allTerminal = false
					break
				}
			}
			if allTerminal && !allOk {
				return false, fmt.Errorf("all jobs are in terminal states and conditions aren't met")
			}

			return allOk, nil

		},
	}
	return waiter.Wait()
}

func WaitForJob(
	ctx context.Context,
	jobID string,
	job executor.Job,

	checkJobStateFunctions ...CheckJobStatesFunction,
) error {
	_, finalJobState := pjob.GetCurrentJobState(job)
	if finalJobState.Status == "Complete" {
		return nil
	}

	if finalJobState.Status != "Complete" {

		return WaitForJobWithLogs(ctx, jobID, false, finalJobState, checkJobStateFunctions...)
	}
	return nil
}

func WaitForJobAllHaveState(nodeIDs []string, states ...executor.JobStateType) CheckJobStatesFunction {

	return func(jobStates map[string]executor.JobStateType) (bool, error) {
		if states[0].String() != "Complete" {

			log.Trace().Msgf("WaitForJobShouldHaveStates:\nnodeIds = %+v,\nstate = %s\njobStates = %+v", nodeIDs, states, jobStates)
			fmt.Printf("WaitForJobShouldHaveStates:\nnodeIds = %+v,\nstate = %s\njobStates = %+v", nodeIDs, states[0], jobStates)
			if len(jobStates) != len(nodeIDs) {
				return false, nil
			}
			seenAll := true
			for _, nodeID := range nodeIDs {
				seenState, ok := jobStates[nodeID]
				isComplete := states[0].String()
				if isComplete == "Complete" {
					break
				}
				if !ok {
					seenAll = false
				} else if !system.StringArrayContains(
					system.GetJobStateStringArray(states), seenState.String()) {
					seenAll = false
				}
			}
			return seenAll, nil

		}
		return false, nil
	}
}

func WaitForJobThrowErrors(job executor.Job, errorStates []executor.JobStateType) CheckJobStatesFunction {

	return func(jobStates map[string]executor.JobStateType) (bool, error) {
		var Status string
		for _, status := range jobStates {
			if status.String() == "Complete" {
				Status = "Complete"
			}
		}
		fmt.Printf("\nStatus %s\n", Status)
		if Status == "Complete" {
			return true, nil

		}
		if Status != "Complete" {

			log.Trace().Msgf("WaitForJobThrowErrors:\nerrorStates = %+v,\njobStates = %+v", errorStates, jobStates)
			fmt.Printf("WaitForJobThrowErrors:\nerrorStates = %+v,\njobStates = %+v", errorStates, jobStates)
			for id, state := range jobStates {
				fmt.Printf("WaitForJobThrowErrors loop: %#v\n %#v\n ", state, state.String())
				if state.String() == "Complete" {
					break

				}
				if system.StringArrayContains(system.GetJobStateStringArray(errorStates), state.String()) && state.String() != "BidRejected" {
					return false, fmt.Errorf("job %s has error state: %s", id, state.String())
				}
			}
		}
		return true, nil

	}
}

func Get(jobID string, timeout int) map[string]bool {
	fmt.Print(timeout)
	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	log.Info().Msgf("Fetching results of job '%s'...", jobID)
	job, ok, err := getAPIClient().Get(context.Background(), jobID)
	if err != nil {
		fmt.Printf("%s", err)
	}
	if !ok {
		fmt.Print("job not found")
	}

	resultCIDs := map[string]bool{}
	for _, jobState := range job.State {
		if jobState.ResultsID != "" {
			resultCIDs[jobState.ResultsID] = true
		}
	}
	log.Debug().Msgf("Job has result CIDs: %v", resultCIDs)

	if len(resultCIDs) == 0 {
		log.Info().Msg("Job has no results.")
		return nil
	}

	swarmAddrs := []string{}
	if getCmdFlags.ipfsSwarmAddrs != "" {
		swarmAddrs = strings.Split(getCmdFlags.ipfsSwarmAddrs, ",")
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	log.Debug().Msg("Spinning up IPFS node...")
	n, err := ipfs.NewNode(cm, swarmAddrs)
	if err != nil {
		fmt.Printf("%s", err)
	}

	log.Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client()
	if err != nil {
		fmt.Printf("%s", err)
	}

	// NOTE: this will run in non-deterministic order
	for cid := range resultCIDs {
		outputDir := filepath.Join(".", cid)
		ok, err := system.PathExists(outputDir)
		if err != nil {
			fmt.Printf("%s", err)
		}
		if ok {
			log.Warn().Msgf("Output directory '%s' already exists, skipping CID '%s'.", outputDir, cid)
			continue
		}

		log.Info().Msgf("Downloading result CID '%s' to '%s'...",
			cid, outputDir)

		ctx, cancel := context.WithDeadline(context.Background(),
			time.Now().Add(time.Second*time.Duration(timeout)))
		defer cancel()

		err = cl.Get(ctx, cid, outputDir)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				msg := fmt.Sprintf("Timed out while downloading result %v", timeout)
				log.Error().Msg(msg)
			}

		}
	}

	return resultCIDs
}

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	dockerCmd.AddCommand(dockerRunCmd)

	// TODO: don't make jobEngine specifiable in the docker subcommand
	dockerRunCmd.PersistentFlags().StringVar(
		&jobEngine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobVerifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputs, "inputs", "i", []string{},
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputUrls, "input-urls", "u", []string{},
		`URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '--input-urls http://foo.com/bar.tar.gz:/app/bar.tar.gz' mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz').`, // nolint:lll // Documentation, ok if long.
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputVolumes, "input-volumes", "v", []string{},
		`CID:path of the input data volumes, if you need to set the path of the mounted data.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobOutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes. 'outputs:/outputs' is always added.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	dockerRunCmd.PersistentFlags().IntVarP(
		&jobConcurrency, "concurrency", "c", 1,
		`How many nodes should run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobCPU, "cpu", "",
		`Job CPU cores (e.g. 500m, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobMemory, "memory", "",
		`Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobGPU, "gpu", "",
		`Job GPU requirement (e.g. 1, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(&jobLabels,
		"labels", "l", []string{},
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, // nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVarP(
		&waitForJobToFinishAndPrintOutput, "wait", "w", false,
		`Wait For Job To Finish And Print Output`,
	)

	// ipfs get wait time
	dockerRunCmd.PersistentFlags().IntVarP(
		&jobIpfsGetTimeOut, "gettimeout", "g", 10,
		`Timeout for getting the results of a job in --wait`,
	)

}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Run a docker job on the network (see run subcommand)",
}

var dockerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a docker job on the network",
	Args:  cobra.MinimumNArgs(1),
	PostRun: func(cmd *cobra.Command, args []string) {
		// Can't think of any reason we'd want these to persist.
		// The below is to clean out for testing purposes. (Kinda ugly to put it in here,
		// but potentially cleaner than making things public, which would
		// be the other way to attack this.)
		jobInputs = []string{}
		jobInputUrls = []string{}
		jobInputVolumes = []string{}
		jobOutputVolumes = []string{}
		jobEnv = []string{}
		jobLabels = []string{}

		jobEngine = "docker"
		jobVerifier = "ipfs"
		jobConcurrency = 1
		jobCPU = ""
		jobMemory = ""
		jobGPU = ""
		skipSyntaxChecking = false
		waitForJobToFinishAndPrintOutput = false
		jobIpfsGetTimeOut = 10
	},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		ctx := context.Background()
		jobImage := cmdArgs[0]
		jobEntrypoint := cmdArgs[1:]

		engineType, err := executor.ParseEngineType(jobEngine)
		if err != nil {
			return err
		}

		verifierType, err := verifier.ParseVerifierType(jobVerifier)
		if err != nil {
			return err
		}

		for _, i := range jobInputs {
			jobInputVolumes = append(jobInputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		jobOutputVolumes = append(jobOutputVolumes, "outputs:/outputs")

		// No error checking, because it will never be an error (for now)
		sanitizationMsgs, sanitizationFatal := system.SanitizeImageAndEntrypoint(jobEntrypoint)
		if sanitizationFatal {
			log.Error().Msgf("Errors: %+v", sanitizationMsgs)
			return fmt.Errorf("could not continue with errors")
		}

		if len(sanitizationMsgs) > 0 {
			log.Warn().Msgf("Found the following possible errors in arguments: %+v", sanitizationMsgs)
		}

		spec, deal, err := pjob.ConstructDockerJob(
			engineType,
			verifierType,
			jobCPU,
			jobMemory,
      		jobGPU,
			jobInputUrls,
			jobInputVolumes,
			jobOutputVolumes,
			jobEnv,
			jobEntrypoint,
			jobImage,
			jobConcurrency,
			jobLabels,
		)

		if err != nil {
			return err
		}

		if !skipSyntaxChecking {
			err = system.CheckBashSyntax(jobEntrypoint)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(ctx, spec, deal, nil)
		if err != nil {
			return err
		}

		states, err := getAPIClient().GetExecutionStates(ctx, job.ID)
		if err != nil {
			return err
		}

		cmd.Printf("%s\n", job.ID)
		currentNodeID, _ := pjob.GetCurrentJobState(job)
		nodeIds := []string{currentNodeID}
		if waitForJobToFinishAndPrintOutput {
			err = WaitForJob(ctx, job.ID, job,
				WaitForJobThrowErrors(job, []executor.JobStateType{
					executor.JobStateCancelled,
					executor.JobStateError,
				}),
				WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
			)
			if err != nil {
				return err
			}

			cidl := Get(job.ID, jobIpfsGetTimeOut)
			var cidv string
			for cid := range cidl {
				cidv = cid
			}
			body, err := ioutil.ReadFile(cidv + "/stdout")
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Println(string(body))
		}

		return nil
	},
}
