package bacalhau

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/executor/wasm"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage/inline"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const null rune = 0

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
	wasmJob := defaultWasmJobSpec()
	runtimeSettings := NewRunTimeSettings()
	downloadSettings := util.NewDownloadSettings()
	var nodeSelector string

	runWasmCommand := &cobra.Command{
		Use:     "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
		Short:   "Run a WASM job on the network",
		Long:    languageRunLong,
		Example: languageRunExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWasm(cmd, args, wasmJob, runtimeSettings, downloadSettings, nodeSelector)
		},
	}

	settingsFlags := NewRunTimeSettingsFlags(runtimeSettings)
	runWasmCommand.Flags().AddFlagSet(settingsFlags)

	downloadFlags := NewIPFSDownloadFlags(downloadSettings)
	runWasmCommand.Flags().AddFlagSet(downloadFlags)

	runWasmCommand.PersistentFlags().StringVarP(
		&nodeSelector, "selector", "s", nodeSelector,
		`Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.`, //nolint:lll // Documentation, ok if long.
	)

	runWasmCommand.PersistentFlags().Var(
		VerifierFlag(&wasmJob.Spec.Verifier), "verifier",
		`What verification engine to use to run the job`,
	)
	runWasmCommand.PersistentFlags().Var(
		PublisherFlag(&wasmJob.Spec.Publisher), "publisher",
		`What publisher engine to use to publish the job results`,
	)
	runWasmCommand.PersistentFlags().IntVarP(
		&wasmJob.Spec.Deal.Concurrency, "concurrency", "c", wasmJob.Spec.Deal.Concurrency,
		`How many nodes should run the job`,
	)
	runWasmCommand.PersistentFlags().IntVar(
		&wasmJob.Spec.Deal.Confidence, "confidence", wasmJob.Spec.Deal.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	runWasmCommand.PersistentFlags().IntVar(
		&wasmJob.Spec.Deal.MinBids, "min-bids", wasmJob.Spec.Deal.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	runWasmCommand.PersistentFlags().Float64Var(
		&wasmJob.Spec.Timeout, "timeout", wasmJob.Spec.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms)`,
	)
	runWasmCommand.PersistentFlags().StringVar(
		&wasmJob.Spec.Wasm.EntryPoint, "entry-point", wasmJob.Spec.Wasm.EntryPoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)
	runWasmCommand.PersistentFlags().VarP(
		NewURLStorageSpecArrayFlag(&wasmJob.Spec.Inputs), "input-urls", "u",
		`URL of the input data volumes downloaded from a URL source. Mounts data at '/inputs' (e.g. '-u http://foo.com/bar.tar.gz'
		mounts 'bar.tar.gz' at '/inputs/bar.tar.gz'). URL accept any valid URL supported by the 'wget' command,
		and supports both HTTP and HTTPS.`,
	)
	runWasmCommand.PersistentFlags().VarP(
		NewIPFSStorageSpecArrayFlag(&wasmJob.Spec.Inputs), "input-volumes", "v",
		`CID:path of the input data volumes, if you need to set the path of the mounted data.`,
	)
	runWasmCommand.PersistentFlags().VarP(
		EnvVarMapFlag(&wasmJob.Spec.Wasm.EnvironmentVariables), "env", "e",
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	runWasmCommand.PersistentFlags().VarP(
		NewURLStorageSpecArrayFlag(&wasmJob.Spec.Wasm.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	runWasmCommand.PersistentFlags().VarP(
		NewIPFSStorageSpecArrayFlag(&wasmJob.Spec.Wasm.ImportModules), "import-module-volumes", "I",
		`CID:path of the WASM modules to import from IPFS, if you need to set the path of the mounted data.`,
	)

	return runWasmCommand
}

func runWasm(
	cmd *cobra.Command,
	args []string,
	wasmJob *model.Job,
	runtimeSettings *RunTimeSettings,
	downloadSettings *model.DownloaderSettings,
	nodeSelector string,
) error {
	ctx := cmd.Context()
	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

	wasmCidOrPath := args[0]
	wasmJob.Spec.Wasm.Parameters = args[1:]

	nodeSelectorRequirements, err := job.ParseNodeSelector(nodeSelector)
	if err != nil {
		return err
	}
	wasmJob.Spec.NodeSelectors = nodeSelectorRequirements

	// Try interpreting this as a CID.
	wasmCid, err := cid.Parse(wasmCidOrPath)
	if err == nil {
		// It is a valid CID – proceed to create IPFS context.
		wasmJob.Spec.Wasm.EntryModule = model.StorageSpec{
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
		inlineData, err := storage.Upload(cmd.Context(), info.Name())
		if err != nil {
			return err
		}
		wasmJob.Spec.Wasm.EntryModule = inlineData
	}

	// We can only use a Deterministic verifier if we have multiple nodes running the job
	// If the user has selected a Deterministic verifier (or we are using it by default)
	// then switch back to a Noop Verifier if the concurrency is too low.
	if wasmJob.Spec.Deal.Concurrency <= 1 && wasmJob.Spec.Verifier == model.VerifierDeterministic {
		wasmJob.Spec.Verifier = model.VerifierNoop
	}

	// See wazero.ModuleConfig.WithEnv
	for key, value := range wasmJob.Spec.Wasm.EnvironmentVariables {
		for _, str := range []string{key, value} {
			if str == "" || strings.ContainsRune(str, null) {
				return fmt.Errorf("invalid environment variable %s=%s", key, value)
			}
		}
	}

	return ExecuteJob(ctx, cm, cmd, wasmJob, *runtimeSettings, *downloadSettings)
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
	module, err := wasm.LoadModule(ctx, engine, programPath)
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
