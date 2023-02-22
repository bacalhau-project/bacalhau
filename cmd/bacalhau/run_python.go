package bacalhau

import (
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage/inline"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	languageRunLong = templates.LongDesc(i18n.T(`
		Runs a job by compiling language file to WASM on the node.
		`))

	languageRunExample = templates.Examples(i18n.T(`
		TBD`))
)

// LanguageRunOptions declares the arguments accepted by the `'language' run` command
type LanguageRunOptions struct {
	Deterministic bool     // Execute this job deterministically
	Inputs        []string // Array of input CIDs
	InputUrls     []string // Array of input URLs (will be copied to IPFS)
	InputVolumes  []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes []string // Array of output volumes in 'name:mount point' form
	Env           []string // Array of environment variables
	Concurrency   int      // Number of concurrent jobs to run
	Confidence    int      // Minimum number of nodes that must agree on a verification result
	MinBids       int      // Minimum number of bids that must be received before any are accepted (at random)
	Timeout       float64  // Job execution timeout in seconds
	Labels        []string // Labels for the job on the Bacalhau network (for searching)

	Command          string // Command to execute
	RequirementsPath string // Path for requirements.txt for executing with Python
	ContextPath      string // ContextPath (code) for executing with Python

	// CPU string
	// Memory string
	// GPU string
	// WorkingDir string // Working directory for docker

	RuntimeSettings  RunTimeSettings
	DownloadSettings model.DownloaderSettings

	// ShardingGlobPattern string
	// ShardingBasePath string
	// ShardingBatchSize int
}

func NewLanguageRunOptions() *LanguageRunOptions {
	return &LanguageRunOptions{
		Deterministic:    true,
		Inputs:           []string{},
		InputUrls:        []string{},
		InputVolumes:     []string{},
		OutputVolumes:    []string{},
		Env:              []string{},
		Concurrency:      1,
		Confidence:       0,
		MinBids:          0, // 0 means no minimum before bidding
		Timeout:          DefaultTimeout.Seconds(),
		Labels:           []string{},
		Command:          "",
		RequirementsPath: "",
		ContextPath:      ".",
		RuntimeSettings:  *NewRunTimeSettings(),
		DownloadSettings: *util.NewDownloadSettings(),
	}
}

// TODO: move the adapter code (from wasm to docker) into a wasm executor, so
// that the compute node can verify the job knowing that it was run properly,
// rather than doing the translation in, and thereby trusting, the client (to
// set up the wasm environment to be determinstic)

func newRunPythonCmd() *cobra.Command {
	OLR := NewLanguageRunOptions()

	runPythonCmd := &cobra.Command{
		Use:     "python",
		Short:   "Run a python job on the network",
		Long:    languageRunLong,
		Example: languageRunExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint
			return runPython(cmd, cmdArgs, OLR)
		},
	}

	// determinism flag
	runPythonCmd.PersistentFlags().BoolVar(
		&OLR.Deterministic, "deterministic", OLR.Deterministic,
		`Enforce determinism: run job in a single-threaded wasm runtime with `+
			`no sources of entropy. NB: this will make the python runtime execute`+
			`in an environment where only some libraries are supported, see `+
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
	runPythonCmd.PersistentFlags().IntVar(
		&OLR.MinBids, "min-bids", OLR.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	runPythonCmd.PersistentFlags().Float64Var(
		&OLR.Timeout, "timeout", OLR.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms)`,
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
	runPythonCmd.PersistentFlags().StringSliceVarP(
		&OLR.Labels, "labels", "l", OLR.Labels,
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)

	runPythonCmd.PersistentFlags().AddFlagSet(NewRunTimeSettingsFlags(&OLR.RuntimeSettings))
	runPythonCmd.PersistentFlags().AddFlagSet(NewIPFSDownloadFlags(&OLR.DownloadSettings))
	return runPythonCmd
}

func runPython(cmd *cobra.Command, cmdArgs []string, OLR *LanguageRunOptions) error {
	ctx := cmd.Context()

	cm := cmd.Context().Value(systemManagerKey).(*system.CleanupManager)

	// error if determinism is false
	if !OLR.Deterministic {
		Fatal(cmd, "Determinism=false not supported yet "+
			"(languages only support wasm backend with forced determinism)", 1)
	}

	var programPath string
	if len(cmdArgs) > 0 {
		programPath = cmdArgs[0]
	}

	if OLR.Command == "" && programPath == "" {
		Fatal(cmd, "Please specify an inline command or a path to a python file.", 1)
	}

	for _, i := range OLR.Inputs {
		OLR.InputVolumes = append(OLR.InputVolumes, fmt.Sprintf("%s:/inputs", i))
	}

	language := "python"
	version := "3.10"

	// TODO: #450 These two code paths make me nervous - the fact that we
	// have ConstructLanguageJob and ConstructDockerJob as separate means
	// manually keeping them in sync.
	j, err := job.ConstructLanguageJob(
		OLR.InputVolumes,
		OLR.InputUrls,
		OLR.OutputVolumes,
		[]string{}, // no env vars (yet)
		OLR.Concurrency,
		OLR.Confidence,
		OLR.MinBids,
		OLR.Timeout,
		language,
		version,
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

	if OLR.ContextPath == "." && OLR.RequirementsPath == "" && programPath == "" {
		cmd.Println("no program or requirements specified, not uploading context - set --context-path to full path to force context upload")
		OLR.ContextPath = ""
	}

	if OLR.ContextPath != "" {
		// construct a tar file from the contextPath directory
		// tar + gzip
		cmd.Printf("Uploading %q to server to execute command in context, press Ctrl+C to cancel\n", OLR.ContextPath)
		time.Sleep(1 * time.Second)
		inlineStorage := inline.NewStorage()
		context, cerr := inlineStorage.Upload(ctx, OLR.ContextPath)
		if cerr != nil {
			Fatal(cmd, cerr.Error(), 1)
			return nil
		}
		context.Path = "/job"
		j.Spec.Contexts = append(j.Spec.Contexts, context)
	}

	err = ExecuteJob(ctx, cm, cmd, j, OLR.RuntimeSettings, OLR.DownloadSettings)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error executing job: %s", err), 1)
		return nil
	}

	return nil
}
