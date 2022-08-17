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

	OLR = &LanguageRunOptions{}
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

//nolint:gochecknoinits
func init() {
	// determinism flag
	runPythonCmd.PersistentFlags().BoolVar(
		&OLR.Deterministic, "deterministic", true,
		`Enforce determinism: run job in a single-threaded wasm runtime with `+
			`no sources of entropy. NB: this will make the python runtime execute`+
			`in an environment where only some librarie are supported, see `+
			`https://pyodide.org/en/stable/usage/packages-in-pyodide.html`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Inputs, "inputs", "i", []string{},
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.InputVolumes, "input-volumes", "v", []string{},
		`CID:path of the input data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.OutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes`,
	)
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Env, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	// TODO: concurrency should be factored out (at least up to run, maybe
	// shared with docker and wasm raw commands too)
	runPythonCmd.PersistentFlags().IntVar(
		&OLR.Concurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
	runPythonCmd.PersistentFlags().StringVarP(
		&OLR.Command, "command", "c", "",
		`Program passed in as string (like python)`,
	)
	runPythonCmd.PersistentFlags().StringVarP(
		&OLR.RequirementsPath, "requirement", "r", "",
		`Install from the given requirements file. (like pip)`, // TODO: This option can be used multiple times.
	)
	runPythonCmd.PersistentFlags().StringVar(
		// TODO: consider replacing this with context-glob, default to
		// "./**/*.py|./requirements.txt", OR .bacalhau_ignore
		&OLR.ContextPath, "context-path", ".",
		"Path to context (e.g. python code) to send to server (via public IPFS network) "+
			"for execution (max 10MiB). Set to empty string to disable",
	)
	runPythonCmd.PersistentFlags().StringVar(
		&OLR.Verifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
	)

	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Labels, "labels", "l", []string{},
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
			return fmt.Errorf("must specify an inline command or a path to a python file")
		}

		for _, i := range ODR.Inputs {
			OLR.InputVolumes = append(OLR.InputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		//nolint:lll // it's ok to be long
		// TODO: #450 These two code paths make me nervous - the fact that we have ConstructLanguageJob and ConstructDockerJob as separate means manually keeping them in sync.
		spec, deal, err := job.ConstructLanguageJob(
			OLR.InputVolumes,
			OLR.InputUrls,
			OLR.OutputVolumes,
			[]string{}, // no env vars (yet)
			OLR.Concurrency,
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
			log.Info().Msgf("no program or requirements specified, not uploading context - set --context-path to full path to force context upload")
			OLR.ContextPath = ""
		}

		if OLR.ContextPath != "" {
			// construct a tar file from the contextPath directory
			// tar + gzip
			log.Info().Msgf("uploading %s to server to execute command in context, press Ctrl+C to cancel", OLR.ContextPath)
			time.Sleep(1 * time.Second)
			err = compress(OLR.ContextPath, &buf)
			if err != nil {
				return err
			}

			// check size of buf
			if buf.Len() > 10*1024*1024 {
				return fmt.Errorf("context tar file is too large (>10MiB)")
			}

		}

		ctx := context.Background()
		job, err := getAPIClient().Submit(ctx, spec, deal, &buf)
		if err != nil {
			return err
		}

		log.Debug().Msgf(
			"submitting job with spec %+v", spec)

		cmd.Printf("%s\n", job.ID)
		return nil
	},
}

// from https://github.com/mimoo/eureka/blob/master/folders.go under Apache 2

//nolint:gocyclo
func compress(src string, buf io.Writer) error {
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
