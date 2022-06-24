package bacalhau

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var deterministic bool
var command string

func init() {
	// determinism flag
	runPythonCmd.PersistentFlags().BoolVar(
		&deterministic, "deterministic", true,
		`Enforce determinism: run job in a single-threaded wasm runtime with `+
			`no sources of entropy. NB: this will make the python runtime execute`+
			`in a wasm environment where only some librarie are supported, see `+
			`https://pyodide.org/en/stable/usage/packages-in-pyodide.html`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&jobInputVolumes, "input-volumes", "v", []string{},
		`cid:path of the input data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&jobOutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	// TODO: concurrency should be factored out (at least up to run, maybe
	// shared with docker and wasm raw commands too)
	runPythonCmd.PersistentFlags().IntVar(
		&jobConcurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
	runPythonCmd.PersistentFlags().StringVarP(
		&command, "command", "c", "",
		`Program passed in as string`,
	)
	runPythonCmd.PersistentFlags().StringVar(
		&jobVerifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
	)
}

// TODO: move the adapter code (from wasm to docker) into a wasm executor, so
// that the compute node can verify the job knowing that it was run properly,
// rather than doing the translation in, and thereby trusting, the client (to
// set up the wasm environment to be determinstic)

var runPythonCmd = &cobra.Command{
	Use:   "python",
	Short: "Run a python job on the network",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint

		// error if determinism is false
		if !deterministic {
			return fmt.Errorf("determinism=false not supported yet " +
				"(python only supports wasm backend with forced determinism)")
		}

		var engineType executor.EngineType
		if deterministic {
			engineType = executor.EngineWasm
		} else {
			engineType = executor.EngineDocker
		}

		verifierType := verifier.VerifierIpfs // this does nothing right now?

		if engineType == executor.EngineWasm {

			// pythonFile := cmdArgs[0]
			// TODO: expose python file on ipfs
			jobImage := ""
			jobEntrypoint := []string{}

			spec, deal, err := job.ConstructJob(
				engineType,
				verifierType,
				jobInputVolumes,
				jobOutputVolumes,
				jobEnv,
				jobEntrypoint,
				jobImage,
				jobConcurrency,
			)
			if err != nil {
				return err
			}

			ctx := context.Background()
			job, err := getAPIClient().Submit(ctx, spec, deal)
			if err != nil {
				return err
			}

			log.Debug().Msgf(
				"submitting job with spec %+v", spec)

			fmt.Printf("%s\n", job.Id)
		}
		return nil
	},
}
