package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/opts"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	storagespec "github.com/bacalhau-project/bacalhau/pkg/model/spec"
	wasm2 "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
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
const defaultEntrypoint = "_start"

type WasmRunOptions struct {
	Verifier             model.Verifier // Verifier - verifier.Verifier
	Concurrency          int            // Number of concurrent jobs to run
	Confidence           int            // Minimum number of nodes that must agree on a verification result
	MinBids              int            // Minimum number of bids before they will be accepted (at random)
	Timeout              float64        // Job execution timeout in seconds
	Entrypoint           string
	EnvironmentVariables map[string]string
	ImportModules        []model.StorageSpec
	RunTimeSettings      RunTimeSettings
	DownloadFlags        model.DownloaderSettings
	NodeSelector         string // Selector (label query) to filter nodes on which this job can be executed
	Publisher            opts.PublisherOpt
	Inputs               opts.StorageOpt
	Outputs              opts.StorageOpt
}

func NewRunWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		RunTimeSettings: *NewRunTimeSettings(),
		DownloadFlags:   *util.NewDownloadSettings(),
		Publisher:       opts.NewPublisherOptFromSpec(model.PublisherSpec{Type: model.PublisherEstuary}),
		Verifier:        model.VerifierDeterministic,
		Timeout:         DefaultTimeout.Seconds(),
		Entrypoint:      defaultEntrypoint,
		Concurrency:     1,
	}
}

// TODO now unused
/*
func defaultWasmJobSpec() *model.Job {
	wasmEngine, err := (&wasm2.WasmEngineSpec{
		EntryPoint: defaultEntrypoint,
	}).AsSpec()
	if err != nil {
		panic(err)
	}
	wasmJob, _ := model.NewJobWithSaneProductionDefaults()
	wasmJob.Spec.Engine = wasmEngine
	wasmJob.Spec.Verifier = model.VerifierDeterministic
	wasmJob.Spec.Timeout = DefaultTimeout.Seconds()
	wasmJob.Spec.Outputs = []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	}

	return wasmJob
}
*/

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
		VerifierFlag(&ODR.Verifier), "verifier",
		`What verification engine to use to run the job`,
	)
	wasmRunCmd.PersistentFlags().VarP(&ODR.Publisher, "publisher", "p",
		`Where to publish the result of the job`,
	)
	wasmRunCmd.PersistentFlags().IntVarP(
		&ODR.Concurrency, "concurrency", "c", ODR.Concurrency,
		`How many nodes should run the job`,
	)
	wasmRunCmd.PersistentFlags().IntVar(
		&ODR.Confidence, "confidence", ODR.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	wasmRunCmd.PersistentFlags().IntVar(
		&ODR.MinBids, "min-bids", ODR.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	wasmRunCmd.PersistentFlags().Float64Var(
		&ODR.Timeout, "timeout", ODR.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms)`,
	)
	wasmRunCmd.PersistentFlags().StringVar(
		&ODR.Entrypoint, "entry-point", ODR.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)
	wasmRunCmd.PersistentFlags().VarP(&ODR.Inputs, "input", "i", inputUsageMsg)
	wasmRunCmd.PersistentFlags().VarP(
		EnvVarMapFlag(&ODR.EnvironmentVariables), "env", "e",
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	wasmRunCmd.PersistentFlags().VarP(
		NewURLStorageSpecArrayFlag(&ODR.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	wasmRunCmd.PersistentFlags().VarP(
		NewIPFSStorageSpecArrayFlag(&ODR.ImportModules), "import-module-volumes", "I",
		`CID:path of the WASM modules to import from IPFS, if you need to set the path of the mounted data.`,
	)

	return wasmRunCmd
}

// TODO find a home for this.
func FlattenMap(inputMap map[string]string) []string {
	var result []string
	for key, value := range inputMap {
		result = append(result, key)
		result = append(result, value)
	}
	return result
}

func createWasmJob(ctx context.Context, cmd *cobra.Command, cmdArgs []string, opts *WasmRunOptions) (*model.WasmJob, error) {
	nodeSelectors, err := job.ParseNodeSelector(opts.NodeSelector)
	if err != nil {
		return nil, err
	}
	wasmCidOrPath := cmdArgs[0]
	// Try interpreting this as a CID.
	var entryModule storagespec.Storage
	wasmCid, err := cid.Parse(wasmCidOrPath)
	if err == nil {
		// It is a valid CID – proceed to create IPFS context.
		// TODO again does wasm entry module need to be different? It doesn't have a name or a mount.
		entryModule, err = (&ipfs.IPFSStorageSpec{CID: wasmCid}).AsSpec("TODO", "TODO")
		if err != nil {
			return nil, err
		}
	} else {
		// Try interpreting this as a path.
		info, err := os.Stat(wasmCidOrPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "%q is not a valid CID or local file", wasmCidOrPath)
			} else {
				return nil, err
			}
		}

		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%q should point to a single file", wasmCidOrPath)
		}

		err = os.Chdir(filepath.Dir(wasmCidOrPath))
		if err != nil {
			return nil, err
		}

		cmd.Printf("Uploading %q to server to execute command in context, press Ctrl+C to cancel\n", wasmCidOrPath)
		// TODO this is a hilariously short duration to wait for user input. I'd rather abandon it, or reverse the logic to require input to continue.
		time.Sleep(1 * time.Second)

		inlineStorage := inline.NewStorage()
		inlineData, err := inlineStorage.Upload(ctx, info.Name())
		if err != nil {
			return nil, err
		}
		entryModule = inlineData
	}

	// We can only use a Deterministic verifier if we have multiple nodes running the job
	// If the user has selected a Deterministic verifier (or we are using it by default)
	// then switch back to a Noop Verifier if the concurrency is too low.
	var verifier model.Verifier
	if opts.Concurrency <= 1 && opts.Verifier == model.VerifierDeterministic {
		verifier = model.VerifierNoop
	}

	// See wazero.ModuleConfig.WithEnv
	for key, value := range opts.EnvironmentVariables {
		for _, str := range []string{key, value} {
			if str == "" || strings.ContainsRune(str, null) {
				return nil, fmt.Errorf("invalid environment variable %s=%s", key, value)
			}
		}
	}

	out := &model.WasmJob{
		// TODO this could be different than the api version as it only relates to docker jobs.
		APIVersion: model.APIVersionLatest(),
		WasmSpec: wasm2.WasmEngineSpec{
			EntryModule:          entryModule,
			EntryPoint:           opts.Entrypoint,
			Parameters:           cmdArgs[1:],
			EnvironmentVariables: FlattenMap(opts.EnvironmentVariables),
			// TODO not sure these are used yet.
			//ImportModules:        opts.ImportModules,
		},
		PublisherSpec:  opts.Publisher.Value(),
		VerifierSpec:   verifier,
		ResourceConfig: model.ResourceUsageConfig{
			// TODO
			/*
				CPU:    opts.CPU,
				Memory: opts.Memory,
				Disk:   opts.Disk,
				GPU:    opts.GPU,

			*/
		},
		NetworkConfig: model.NetworkConfig{
			// TODO
			/*
				Type:    opts.Networking,
				Domains: opts.NetworkDomains,
			*/
		},
		Inputs:  opts.Inputs.Values(),
		Outputs: opts.Outputs.Values(),
		DealSpec: model.Deal{
			Concurrency: opts.Concurrency,
			Confidence:  opts.Confidence,
			MinBids:     opts.MinBids,
		},
		NodeSelectors: nodeSelectors,
		Timeout:       opts.Timeout,
		// TODO
		//Annotations:   opts.Labels,
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

func runWasm(cmd *cobra.Command, args []string, opts *WasmRunOptions) error {
	ctx := cmd.Context()
	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

	wasmJob, err := createWasmJob(ctx, cmd, args, opts)
	if err != nil {
		return err
	}

	return ExecuteWasmJob(ctx, cm, cmd, wasmJob, &ExecutionSettings{
		Runtime:  opts.RunTimeSettings,
		Download: opts.DownloadFlags,
	})
}

func newValidateWasmCmd() *cobra.Command {
	entryPoint := defaultEntrypoint

	validateWasmCommand := &cobra.Command{
		Use:   "validate <local.wasm> [--entry-point <string>]",
		Short: "Check that a WASM program is runnable on the network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateWasm(cmd, args, entryPoint)
		},
	}

	validateWasmCommand.PersistentFlags().StringVar(
		&entryPoint, "entry-point", defaultEntrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	return validateWasmCommand
}

func validateWasm(cmd *cobra.Command, args []string, entrypoint string) error {
	ctx := cmd.Context()

	programPath := args[0]

	engine := wazero.NewRuntime(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "engine", engine)

	config := wazero.NewModuleConfig()
	storage := model.NewNoopProvider[cid.Cid, storage.Storage](noop.NewNoopStorage())
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

	err = wasm.ValidateModuleAsEntryPoint(module, entrypoint)
	if err != nil {
		Fatal(cmd, err.Error(), 2)
		return err
	}

	cmd.Println("OK")
	return nil
}
