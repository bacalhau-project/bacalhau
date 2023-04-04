package bacalhau

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/opts"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	wasmRunLong = templates.LongDesc(i18n.T(`
		Runs a job that was compiled to WASM
		`))

	wasmRunExample = templates.Examples(i18n.T(`
		# Runs the <localfile.wasm> module in bacalhau
		bacalhau wasm run <localfile.wasm>

		# Fetches the wasm module from <cid> and executes it.
		bacalhau wasm run <cid>
		`))
)

const null rune = 0

type WasmRunOptions struct {
	Job             *model.Job
	RunTimeSettings RunTimeSettings
	DownloadFlags   model.DownloaderSettings
	NodeSelector    string // Selector (label query) to filter nodes on which this job can be executed
	Inputs          opts.StorageOpt
}

func NewRunWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		Job:             defaultWasmJobSpec(),
		RunTimeSettings: *NewRunTimeSettings(),
		DownloadFlags:   *util.NewDownloadSettings(),
		NodeSelector:    "",
		Inputs:          opts.StorageOpt{},
	}
}

func defaultWasmJobSpec() *model.Job {
	wasmJob, _ := model.NewJobWithSaneProductionDefaults()
	wasmJob.Spec.Engine = model.EngineWasm
	wasmJob.Spec.Verifier = model.VerifierDeterministic
	wasmJob.Spec.Timeout = DefaultTimeout.Seconds()
	wasmJob.Spec.Wasm.EntryPoint = "_start"
	wasmJob.Spec.Wasm.EnvironmentVariables = map[string]string{}
	wasmJob.Spec.Outputs = []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	}

	return wasmJob
}

func newWasmCmd() *cobra.Command {
	wasmCmd := &cobra.Command{
		Use:               "wasm",
		Short:             "Run and prepare WASM jobs on the network",
		PersistentPreRunE: checkVersion,
	}

	wasmCmd.AddCommand(
		newRunWasmCmd(),
		newValidateWasmCmd(),
	)

	return wasmCmd
}

func newRunWasmCmd() *cobra.Command {
	ODR := NewRunWasmOptions()

	wasmRunCmd := &cobra.Command{
		Use:     "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
		Short:   "Run a WASM job on the network",
		Long:    wasmRunLong,
		Example: wasmRunExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWasm(cmd, args, ODR)
		},
	}

	wasmRunCmd.PersistentFlags().AddFlagSet(NewRunTimeSettingsFlags(&ODR.RunTimeSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(NewIPFSDownloadFlags(&ODR.DownloadFlags))

	wasmRunCmd.PersistentFlags().StringVarP(
		&ODR.NodeSelector, "selector", "s", ODR.NodeSelector,
		`Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.`, //nolint:lll // Documentation, ok if long.
	)

	wasmRunCmd.PersistentFlags().Var(
		VerifierFlag(&ODR.Job.Spec.Verifier), "verifier",
		`What verification engine to use to run the job`,
	)
	wasmRunCmd.PersistentFlags().Var(
		PublisherFlag(&ODR.Job.Spec.Publisher), "publisher",
		`What publisher engine to use to publish the job results`,
	)
	wasmRunCmd.PersistentFlags().IntVarP(
		&ODR.Job.Spec.Deal.Concurrency, "concurrency", "c", ODR.Job.Spec.Deal.Concurrency,
		`How many nodes should run the job`,
	)
	wasmRunCmd.PersistentFlags().IntVar(
		&ODR.Job.Spec.Deal.Confidence, "confidence", ODR.Job.Spec.Deal.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	wasmRunCmd.PersistentFlags().IntVar(
		&ODR.Job.Spec.Deal.MinBids, "min-bids", ODR.Job.Spec.Deal.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	wasmRunCmd.PersistentFlags().Float64Var(
		&ODR.Job.Spec.Timeout, "timeout", ODR.Job.Spec.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms)`,
	)
	wasmRunCmd.PersistentFlags().StringVar(
		&ODR.Job.Spec.Wasm.EntryPoint, "entry-point", ODR.Job.Spec.Wasm.EntryPoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)
	wasmRunCmd.PersistentFlags().VarP(&ODR.Inputs, "input", "i", inputUsageMsg)
	wasmRunCmd.PersistentFlags().VarP(
		EnvVarMapFlag(&ODR.Job.Spec.Wasm.EnvironmentVariables), "env", "e",
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	wasmRunCmd.PersistentFlags().VarP(
		NewURLStorageSpecArrayFlag(&ODR.Job.Spec.Wasm.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	wasmRunCmd.PersistentFlags().VarP(
		NewIPFSStorageSpecArrayFlag(&ODR.Job.Spec.Wasm.ImportModules), "import-module-volumes", "I",
		`CID:path of the WASM modules to import from IPFS, if you need to set the path of the mounted data.`,
	)

	return wasmRunCmd
}

func runWasm(
	cmd *cobra.Command,
	args []string,
	ODR *WasmRunOptions,
) error {
	ctx := cmd.Context()
	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

	wasmCidOrPath := args[0]
	ODR.Job.Spec.Wasm.Parameters = args[1:]

	nodeSelectorRequirements, err := job.ParseNodeSelector(ODR.NodeSelector)
	if err != nil {
		return err
	}
	ODR.Job.Spec.NodeSelectors = nodeSelectorRequirements
	ODR.Job.Spec.Inputs = ODR.Inputs.Values()

	// Try interpreting this as a CID.
	wasmCid, err := cid.Parse(wasmCidOrPath)
	if err == nil {
		// It is a valid CID – proceed to create IPFS context.
		ODR.Job.Spec.Wasm.EntryModule = model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           wasmCid.String(),
		}
	} else {
		// Try interpreting this as a path.
		info, err := os.Stat(wasmCidOrPath)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.Wrapf(err, "%q is not a valid CID or local file", wasmCidOrPath)
			} else {
				return err
			}
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("%q should point to a single file", wasmCidOrPath)
		}

		err = os.Chdir(filepath.Dir(wasmCidOrPath))
		if err != nil {
			return err
		}

		cmd.Printf("Uploading %q to server to execute command in context, press Ctrl+C to cancel\n", wasmCidOrPath)
		time.Sleep(1 * time.Second)

		storage := inline.NewStorage()
		inlineData, err := storage.Upload(ctx, info.Name())
		if err != nil {
			return err
		}
		ODR.Job.Spec.Wasm.EntryModule = inlineData
	}

	// We can only use a Deterministic verifier if we have multiple nodes running the job
	// If the user has selected a Deterministic verifier (or we are using it by default)
	// then switch back to a Noop Verifier if the concurrency is too low.
	if ODR.Job.Spec.Deal.Concurrency <= 1 && ODR.Job.Spec.Verifier == model.VerifierDeterministic {
		ODR.Job.Spec.Verifier = model.VerifierNoop
	}

	// See wazero.ModuleConfig.WithEnv
	for key, value := range ODR.Job.Spec.Wasm.EnvironmentVariables {
		for _, str := range []string{key, value} {
			if str == "" || strings.ContainsRune(str, null) {
				return fmt.Errorf("invalid environment variable %s=%s", key, value)
			}
		}
	}

	return ExecuteJob(ctx, cm, cmd, ODR.Job, ODR.RunTimeSettings, ODR.DownloadFlags)
}

func newValidateWasmCmd() *cobra.Command {
	wasmJob := defaultWasmJobSpec()

	validateWasmCommand := &cobra.Command{
		Use:   "validate <local.wasm> [--entry-point <string>]",
		Short: "Check that a WASM program is runnable on the network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateWasm(cmd, args, wasmJob)
		},
	}

	validateWasmCommand.PersistentFlags().StringVar(
		&wasmJob.Spec.Wasm.EntryPoint, "entry-point", wasmJob.Spec.Wasm.EntryPoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	return validateWasmCommand
}

func validateWasm(cmd *cobra.Command, args []string, wasmJob *model.Job) error {
	ctx := cmd.Context()

	programPath := args[0]
	entryPoint := wasmJob.Spec.Wasm.EntryPoint

	engine := wazero.NewRuntime(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "engine", engine)

	config := wazero.NewModuleConfig()
	storage := model.NewNoopProvider[model.StorageSourceType, storage.Storage](noop.NewNoopStorage())
	loader := wasm.NewModuleLoader(engine, config, storage)
	module, err := loader.Load(ctx, programPath)
	if err != nil {
		Fatal(cmd, err.Error(), 1)
		return err
	}

	wasi, err := wasi_snapshot_preview1.NewBuilder(engine).Compile(ctx)
	if err != nil {
		Fatal(cmd, err.Error(), 3)
		return err
	}

	err = wasm.ValidateModuleImports(module, wasi)
	if err != nil {
		Fatal(cmd, err.Error(), 2)
		return err
	}

	err = wasm.ValidateModuleAsEntryPoint(module, entryPoint)
	if err != nil {
		Fatal(cmd, err.Error(), 2)
		return err
	}

	cmd.Println("OK")
	return nil
}
