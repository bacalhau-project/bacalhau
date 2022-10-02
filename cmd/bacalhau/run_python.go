package bacalhau

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	languageRunLong = templates.LongDesc(i18n.T(`
		Runs a job by compiling language file to WASM on the node.
		`))

	languageRunExample = templates.Examples(i18n.T(`
		TBD`))

	OLR = NewLanguageRunOptions()
)

// LanguageRunOptions declares the arguments accepted by the `'language' run` command
type LanguageRunOptions struct {
	Deterministic bool     // Execute this job deterministically
	Verifier      string   // Verifier - verifier.Verifier
	Inputs        []string // Array of input CIDs
	InputUrls     []string // Array of input URLs (will be copied to IPFS)
	InputVolumes  []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes []string // Array of output volumes in 'name:mount point' form
	Env           []string // Array of environment variables
	Concurrency   int      // Number of concurrent jobs to run
	Confidence    int      // Minimum number of nodes that must agree on a verification result
	MinBids       int      // Minimum number of bids that must be received before any are accepted (at random)
	Labels        []string // Labels for the job on the Bacalhau network (for searching)

	Command          string // Command to execute
	RequirementsPath string // Path for requirements.txt for executing with Python
	ContextPath      string // ContextPath (code) for executing with Python

	// CPU string
	// Memory string
	// GPU string
	// WorkingDir string // Working directory for docker

	// WaitForJobToFinish bool // Wait for the job to execute before exiting
	// WaitForJobToFinishAndPrintOutput bool // Wait for the job to execute, and print the results before exiting
	// WaitForJobTimeoutSecs int // Job time out in seconds

	// ShardingGlobPattern string
	// ShardingBasePath string
	// ShardingBatchSize int
}

func NewLanguageRunOptions() *LanguageRunOptions {
	return &LanguageRunOptions{
		Deterministic:    true,
		Verifier:         "ipfs",
		Inputs:           []string{},
		InputUrls:        []string{},
		InputVolumes:     []string{},
		OutputVolumes:    []string{},
		Env:              []string{},
		Concurrency:      1,
		Confidence:       0,
		Labels:           []string{},
		Command:          "",
		RequirementsPath: "",
		ContextPath:      ".",
	}
}

//nolint:gochecknoinits
func init() {
	// determinism flag
	runPythonCmd.PersistentFlags().BoolVar(
		&OLR.Deterministic, "deterministic", OLR.Deterministic,
		`Enforce determinism: run job in a single-threaded wasm runtime with `+
			`no sources of entropy. NB: this will make the python runtime execute`+
			`in an environment where only some librarie are supported, see `+
			`https://pyodide.org/en/stable/usage/packages-in-pyodide.html`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Inputs, "inputs", "i", OLR.Inputs,
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.InputVolumes, "input-volumes", "v", OLR.InputVolumes,
		`CID:path of the input data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.OutputVolumes, "output-volumes", "o", OLR.OutputVolumes,
		`name:path of the output data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Env, "env", "e", OLR.Env,
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	// TODO: concurrency should be factored out (at least up to run, maybe
	// shared with docker and wasm raw commands too)
	runPythonCmd.PersistentFlags().IntVar(
		&OLR.Concurrency, "concurrency", OLR.Concurrency,
		`How many nodes should run the job`,
	)
	runPythonCmd.PersistentFlags().IntVar(
		&OLR.Confidence, "confidence", OLR.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	runPythonCmd.PersistentFlags().StringVarP(
		&OLR.Command, "command", "c", OLR.Command,
		`Program passed in as string (like python)`,
	)
	runPythonCmd.PersistentFlags().StringVarP(
		&OLR.RequirementsPath, "requirement", "r", OLR.RequirementsPath,
		`Install from the given requirements file. (like pip)`, // TODO: This option can be used multiple times.
	)
	runPythonCmd.PersistentFlags().StringVar(
		// TODO: consider replacing this with context-glob, default to
		// "./**/*.py|./requirements.txt", OR .bacalhau_ignore
		&OLR.ContextPath, "context-path", OLR.ContextPath,
		"Path to context (e.g. python code) to send to server (via public IPFS network) "+
			"for execution (max 10MiB). Set to empty string to disable",
	)
	runPythonCmd.PersistentFlags().StringVar(
		&OLR.Verifier, "verifier", OLR.Verifier,
		`What verification engine to use to run the job`,
	)

	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Labels, "labels", "l", OLR.Labels,
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)
}

// TODO: move the adapter code (from wasm to docker) into a wasm executor, so
// that the compute node can verify the job knowing that it was run properly,
// rather than doing the translation in, and thereby trusting, the client (to
// set up the wasm environment to be determinstic)

var runPythonCmd = &cobra.Command{
	Use:     "python",
	Short:   "Run a python job on the network",
	Long:    languageRunLong,
	Example: languageRunExample,
	Args:    cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "cmd/bacalhau/list")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		// error if determinism is false
		if !OLR.Deterministic {
			return fmt.Errorf("determinism=false not supported yet " +
				"(python only supports wasm backend with forced determinism)")
		}

		// TODO: prepare context

		var programPath string
		if len(cmdArgs) > 0 {
			programPath = cmdArgs[0]
		}

		if OLR.Command == "" && programPath == "" {
			Fatal("Please specify an inline command or a path to a python file.", 1)
		}

		for _, i := range ODR.Inputs {
			OLR.InputVolumes = append(OLR.InputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		if len(OLR.InputVolumes) == 0 {
			// TODO: #765 this is a hack to make the job run when no inputs provided
			// Just put a default one down - nothing will be in there.
			OLR.InputVolumes = append(OLR.InputVolumes, "/inputs:/inputs")
		}

		//nolint:lll // it's ok to be long
		// TODO: #450 These two code paths make me nervous - the fact that we have ConstructLanguageJob and ConstructDockerJob as separate means manually keeping them in sync.
		j, err := job.ConstructLanguageJob(
			model.APIVersionLatest(),
			OLR.InputVolumes,
			OLR.InputUrls,
			OLR.OutputVolumes,
			[]string{}, // no env vars (yet)
			OLR.Concurrency,
			OLR.Confidence,
			OLR.MinBids,
			"python",
			"3.10",
			OLR.Command,
			programPath,
			OLR.RequirementsPath,
			OLR.ContextPath,
			OLR.Deterministic,
			OLR.Labels,
			doNotTrack,
		)
		if err != nil {
			return err
		}

		var buf bytes.Buffer

		if OLR.ContextPath == "." && OLR.RequirementsPath == "" && programPath == "" {
			cmd.Println("no program or requirements specified, not uploading context - set --context-path to full path to force context upload")
			OLR.ContextPath = ""
		}

		if OLR.ContextPath != "" {
			// construct a tar file from the contextPath directory
			// tar + gzip
			cmd.Printf("Uploading %s to server to execute command in context, press Ctrl+C to cancel\n", OLR.ContextPath)
			time.Sleep(1 * time.Second)
			err = compress(ctx, OLR.ContextPath, &buf)
			if err != nil {
				return err
			}

			// check size of buf
			if buf.Len() > 10*1024*1024 {
				Fatal("context tar file is too large (>10MiB)", 1)
			}

		}

		log.Debug().Msgf(
			"submitting job %+v", j)

		returnedJob, err := GetAPIClient().Submit(ctx, j, &buf)
		if err != nil {
			Fatal(fmt.Sprintf("Error submitting job: %s", err), 1)
		}

		err = PrintReturnedJobIDToUser(returnedJob)
		if err != nil {
			Fatal(fmt.Sprintf("Error submitting job: %s", err), 1)
		}
		return nil
	},
}

// from https://github.com/mimoo/eureka/blob/master/folders.go under Apache 2

//nolint:gocyclo
func compress(ctx context.Context, src string, buf io.Writer) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/runPython.compress")
	defer span.End()

	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	// is file a folder?
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		// get header
		var header *tar.Header
		header, err = tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		// write header
		if err = tw.WriteHeader(header); err != nil { //nolint:gocritic
			return err
		}
		// get content
		var data *os.File
		data, err = os.Open(src)
		if err != nil {
			return err
		}
		if _, err = io.Copy(tw, data); err != nil {
			return err
		}
	} else if mode.IsDir() { // folder
		// walk through every file in the folder
		err = filepath.Walk(src, func(file string, fi os.FileInfo, _ error) error {
			// generate tar header
			var header *tar.Header
			header, err = tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			// must provide real name
			// (see https://golang.org/src/archive/tar/common.go?#L626)
			header.Name = filepath.ToSlash(file)

			// write header
			if err = tw.WriteHeader(header); err != nil { //nolint:gocritic
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				var data *os.File
				data, err = os.Open(file)
				if err != nil {
					return err
				}
				if _, err = io.Copy(tw, data); err != nil { //nolint:gocritic
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error: file type not supported")
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}
