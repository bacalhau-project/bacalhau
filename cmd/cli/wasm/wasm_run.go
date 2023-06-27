package wasm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
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

type WasmRunOptions struct {
	// parameters and entry modules are arguments
	ImportModules []model.StorageSpec
	Entrypoint    string

	SpecSettings       *flags.SpecFlagSettings       // Setting for top level job spec fields.
	ResourceSettings   *flags.ResourceUsageSettings  // Settings for the jobs resource requirements.
	NetworkingSettings *flags.NetworkingFlagSettings // Settings for the jobs networking.
	DealSettings       *flags.DealFlagSettings       // Settings for the jobs deal.
	RunTimeSettings    *flags.RunTimeSettings        // Settings for running the job.
	DownloadSettings   *flags.DownloaderSettings     // Settings for running Download.

}

func NewWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		ImportModules:      []model.StorageSpec{},
		Entrypoint:         "_start",
		SpecSettings:       flags.NewSpecFlagDefaultSettings(),
		ResourceSettings:   flags.NewDefaultResourceUsageSettings(),
		NetworkingSettings: flags.NewDefaultNetworkingFlagSettings(),
		DealSettings:       flags.NewDefaultDealFlagSettings(),
		DownloadSettings:   flags.NewDefaultDownloaderSettings(),
		RunTimeSettings:    flags.NewDefaultRunTimeSettings(),
	}
}

func NewCmd() *cobra.Command {
	wasmCmd := &cobra.Command{
		Use:               "wasm",
		Short:             "Run and prepare WASM jobs on the network",
		PersistentPreRunE: util.CheckVersion,
	}

	wasmCmd.AddCommand(
		newRunCmd(),
		newValidateCmd(),
	)

	return wasmCmd
}

func newRunCmd() *cobra.Command {
	opts := NewWasmOptions()

	wasmRunCmd := &cobra.Command{
		Use:     "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
		Short:   "Run a WASM job on the network",
		Long:    wasmRunLong,
		Example: wasmRunExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  util.ApplyPorcelainLogLevel,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runWasm(cmd, args, opts); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	wasmRunCmd.PersistentFlags().VarP(
		flags.NewURLStorageSpecArrayFlag(&opts.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	wasmRunCmd.PersistentFlags().VarP(
		flags.NewIPFSStorageSpecArrayFlag(&opts.ImportModules), "import-module-volumes", "I",
		`CID:path of the WASM modules to import from IPFS, if you need to set the path of the mounted data.`,
	)
	wasmRunCmd.PersistentFlags().StringVar(
		&opts.Entrypoint, "entry-point", opts.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	wasmRunCmd.PersistentFlags().AddFlagSet(flags.SpecFlags(opts.SpecSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(flags.DealFlags(opts.DealSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(flags.NewDownloadFlags(opts.DownloadSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(flags.NetworkingFlags(opts.NetworkingSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(flags.ResourceUsageFlags(opts.ResourceSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(flags.NewRunTimeSettingsFlags(opts.RunTimeSettings))

	return wasmRunCmd
}

func runWasm(cmd *cobra.Command, args []string, opts *WasmRunOptions) error {
	ctx := cmd.Context()

	j, err := CreateJob(ctx, args, opts)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	if err := job.VerifyJob(ctx, j); err != nil {
		return fmt.Errorf("verifying job: %w", err)
	}

	if opts.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			return fmt.Errorf("converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	executingJob, err := util.ExecuteJob(ctx, j, opts.RunTimeSettings)
	if err != nil {
		return fmt.Errorf("executing job: %w", err)
	}

	return printer.PrintJobExecution(ctx, executingJob, cmd, opts.DownloadSettings, opts.RunTimeSettings, util.GetAPIClient(ctx))
}

func CreateJob(ctx context.Context, cmdArgs []string, opts *WasmRunOptions) (*model.Job, error) {
	parameters := cmdArgs[1:]

	entryModule, err := parseWasmEntryModule(ctx, cmdArgs[0])
	if err != nil {
		return nil, err
	}

	verifierType, err := model.ParseVerifier(opts.SpecSettings.Verifier)
	if err != nil {
		return nil, err
	}

	outputs, err := parse.JobOutputs(ctx, opts.SpecSettings.OutputVolumes)
	if err != nil {
		return nil, err
	}

	nodeSelectorRequirements, err := parse.NodeSelector(opts.SpecSettings.Selector)
	if err != nil {
		return nil, err
	}

	labels, err := parse.Labels(ctx, opts.SpecSettings.Labels)
	if err != nil {
		return nil, err
	}

	wasmEnvvar, err := parseArrayAsMap(opts.SpecSettings.EnvVar)
	if err != nil {
		return nil, fmt.Errorf("wasm env vars invalid: %w", err)
	}

	spec, err := job.MakeWasmSpec(
		*entryModule, opts.Entrypoint, parameters, wasmEnvvar, opts.ImportModules,
		job.WithVerifier(verifierType),
		job.WithPublisher(opts.SpecSettings.Publisher.Value()),
		job.WithResources(
			opts.ResourceSettings.CPU,
			opts.ResourceSettings.Memory,
			opts.ResourceSettings.Disk,
			opts.ResourceSettings.GPU,
		),
		job.WithNetwork(
			opts.NetworkingSettings.Network,
			opts.NetworkingSettings.Domains,
		),
		job.WithTimeout(opts.SpecSettings.Timeout),
		job.WithInputs(opts.SpecSettings.Inputs.Values()...),
		job.WithOutputs(outputs...),
		job.WithAnnotations(labels...),
		job.WithNodeSelector(nodeSelectorRequirements),
		job.WithDeal(
			opts.DealSettings.TargetingMode,
			opts.DealSettings.Concurrency,
			opts.DealSettings.Confidence,
			opts.DealSettings.MinBids,
		),
	)
	if err != nil {
		return nil, err
	}

	return &model.Job{
		APIVersion: model.APIVersionLatest().String(),
		Spec:       spec,
	}, nil
}

func parseArrayAsMap(inputArray []string) (map[string]string, error) {
	if len(inputArray)%2 != 0 {
		return nil, fmt.Errorf("array must have an even number of elements")
	}

	resultMap := make(map[string]string)
	for i := 0; i < len(inputArray); i += 2 {
		key := inputArray[i]
		value := inputArray[i+1]
		resultMap[key] = value
	}

	return resultMap, nil
}

func parseWasmEntryModule(ctx context.Context, in string) (*model.StorageSpec, error) {
	// Try interpreting this as a CID.
	wasmCid, err := cid.Parse(in)
	if err == nil {
		// It is a valid CID – proceed to create IPFS context.
		// TODO(forrest): doesn't this require a name?
		return &model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           wasmCid.String(),
		}, nil
	}
	// Try interpreting this as a path.
	info, err := os.Stat(in)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "%q is not a valid CID or local file", in)
		} else {
			return nil, err
		}
	}

	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q should point to a single file", in)
	}

	if err := os.Chdir(filepath.Dir(in)); err != nil {
		return nil, err
	}

	storage := inline.NewStorage()
	inlineData, err := storage.Upload(ctx, info.Name())
	if err != nil {
		return nil, err
	}
	return &inlineData, nil
}

func newValidateCmd() *cobra.Command {
	opts := NewWasmOptions()

	validateWasmCommand := &cobra.Command{
		Use:   "validate <local.wasm> [--entry-point <string>]",
		Short: "Check that a WASM program is runnable on the network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateWasm(cmd, args, opts); err != nil {
				util.Fatal(cmd, err, 1)
			}
			return nil
		},
	}

	validateWasmCommand.PersistentFlags().StringVar(
		&opts.Entrypoint, "entry-point", opts.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	return validateWasmCommand
}

func validateWasm(cmd *cobra.Command, args []string, opts *WasmRunOptions) error {
	ctx := cmd.Context()

	programPath := args[0]
	entryPoint := opts.Entrypoint

	engine := wazero.NewRuntime(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "engine", engine)

	config := wazero.NewModuleConfig()
	storage := model.NewNoopProvider[model.StorageSourceType, storage.Storage](noop.NewNoopStorage())
	loader := wasm.NewModuleLoader(engine, config, storage)
	module, err := loader.Load(ctx, programPath)
	if err != nil {
		return err
	}

	wasi, err := wasi_snapshot_preview1.NewBuilder(engine).Compile(ctx)
	if err != nil {
		return err
	}

	err = wasm.ValidateModuleImports(module, wasi)
	if err != nil {
		return err
	}

	err = wasm.ValidateModuleAsEntryPoint(module, entryPoint)
	if err != nil {
		return err
	}

	cmd.Println("OK")
	return nil
}
