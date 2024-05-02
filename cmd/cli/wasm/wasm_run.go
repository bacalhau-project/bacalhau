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
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
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
	ImportModules []*models.InputSource
	Entrypoint    string

	SpecSettings       *cliflags.SpecFlagSettings            // Setting for top level job spec fields.
	ResourceSettings   *cliflags.ResourceUsageSettings       // Settings for the jobs resource requirements.
	NetworkingSettings *cliflags.NetworkingFlagSettings      // Settings for the jobs networking.
	DealSettings       *cliflags.DealFlagSettings            // Settings for the jobs deal.
	RunTimeSettings    *cliflags.RunTimeSettingsWithDownload // Settings for running the job.
	DownloadSettings   *cliflags.DownloaderSettings          // Settings for running Download.

}

func NewWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		ImportModules:      make([]*models.InputSource, 0),
		Entrypoint:         "_start",
		SpecSettings:       cliflags.NewSpecFlagDefaultSettings(),
		ResourceSettings:   cliflags.NewDefaultResourceUsageSettings(),
		NetworkingSettings: cliflags.NewDefaultNetworkingFlagSettings(),
		DealSettings:       cliflags.NewDefaultDealFlagSettings(),
		DownloadSettings:   cliflags.NewDefaultDownloaderSettings(),
		RunTimeSettings:    cliflags.DefaultRunTimeSettingsWithDownload(),
	}
}

func NewCmd() *cobra.Command {
	wasmCmd := &cobra.Command{
		Use:                "wasm",
		Short:              "Run and prepare WASM jobs on the network",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	wasmCmd.AddCommand(
		newRunCmd(),
		newValidateCmd(),
	)

	return wasmCmd
}

func newRunCmd() *cobra.Command {
	opts := NewWasmOptions()

	wasmRunFlags := map[string][]configflags.Definition{
		"ipfs": configflags.IPFSFlags,
	}

	wasmRunCmd := &cobra.Command{
		Use:      "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
		Short:    "Run a WASM job on the network",
		Long:     wasmRunLong,
		Example:  wasmRunExample,
		Args:     cobra.MinimumNArgs(1),
		PreRunE:  hook.Chain(hook.ClientPreRunHooks, configflags.PreRun(wasmRunFlags)),
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWasm(cmd, args, opts)
		},
	}

	wasmRunCmd.PersistentFlags().VarP(
		flags.NewURLStorageSpecArrayFlag(&opts.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	wasmRunCmd.PersistentFlags().StringVar(
		&opts.Entrypoint, "entry-point", opts.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.SpecFlags(opts.SpecSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.DealFlags(opts.DealSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.NewDownloadFlags(opts.DownloadSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.NetworkingFlags(opts.NetworkingSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.ResourceUsageFlags(opts.ResourceSettings))
	wasmRunCmd.PersistentFlags().AddFlagSet(cliflags.NewRunTimeSettingsFlagsWithDownload(opts.RunTimeSettings))

	if err := configflags.RegisterFlags(wasmRunCmd, wasmRunFlags); err != nil {
		util.Fatal(wasmRunCmd, err, 1)
	}

	return wasmRunCmd
}

func runWasm(cmd *cobra.Command, args []string, opts *WasmRunOptions) error {
	ctx := cmd.Context()

	job, err := CreateJobWasm(ctx, args, opts)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	// Normalize and validate the job spec
	job.Normalize()
	if err := job.ValidateSubmission(); err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// TODO(forrest) [refactor]: this options is _almost_ useful. At present it marshals the entire
	// job spec to yaml, said spec cannot be used with `bacalhau job run` since it contains fields that
	// users are not permitted to set, like ID, Version, ModifyTime, State, etc.
	// The solution here is to have a "JobSubmission" type that is different from the actual job spec.
	if opts.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(job)
		if err != nil {
			return fmt.Errorf("converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	api := util.GetAPIClientV2(cmd)
	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	if len(resp.Warnings) > 0 {
		printWarnings(cmd, resp.Warnings)
	}

	if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, &opts.RunTimeSettings.RunTimeSettings, api); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

// TODO(forrest) [refactor]: dedupe from docker_run
func printWarnings(cmd *cobra.Command, warnings []string) {
	cmd.Println("Warnings:")
	for _, warning := range warnings {
		cmd.Printf("\t* %s\n", warning)
	}
}

func CreateJobWasm(ctx context.Context, cmdArgs []string, opts *WasmRunOptions) (*models.Job, error) {
	parameters := cmdArgs[1:]

	entryModule, err := parseWasmEntryModule(ctx, cmdArgs[0])
	if err != nil {
		return nil, err
	}

	envvar, err := parse.StringSliceToMap(opts.SpecSettings.EnvVar)
	if err != nil {
		return nil, fmt.Errorf("wasm env vars invalid: %w", err)
	}
	engineSpec, err := models.WasmSpecBuilder(&models.InputSource{
		Source: entryModule,
		Alias:  "TODO",
		Target: "TODO",
	}).
		WithEntrypoint(opts.Entrypoint).
		WithParameters(parameters...).
		WithEnvironmentVariables(envvar).
		WithImportModules(opts.ImportModules...).
		Build()
	if err != nil {
		return nil, err
	}

	// TODO(forrest) [refactor]: this logic is duplicated in docker_run
	resultPaths := make([]*models.ResultPath, 0, len(opts.SpecSettings.OutputVolumes))
	for name, path := range opts.SpecSettings.OutputVolumes {
		resultPaths = append(resultPaths, &models.ResultPath{
			Name: name,
			Path: path,
		})
	}

	task, err := models.NewTaskBuilder().
		Name("TODO").
		Engine(engineSpec).
		Publisher(opts.SpecSettings.Publisher.Value()).
		ResourcesConfig(&models.ResourcesConfig{
			CPU:    opts.ResourceSettings.CPU,
			Memory: opts.ResourceSettings.Memory,
			Disk:   opts.ResourceSettings.Disk,
			GPU:    opts.ResourceSettings.GPU,
		}).
		InputSources(opts.SpecSettings.Inputs.Values()...).
		ResultPaths(resultPaths...).
		Network(&models.NetworkConfig{
			Type:    opts.NetworkingSettings.Network,
			Domains: opts.NetworkingSettings.Domains,
		}).
		Timeouts(&models.TimeoutConfig{ExecutionTimeout: opts.SpecSettings.Timeout}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	labels, err := parse.StringSliceToMap(opts.SpecSettings.Labels)
	if err != nil {
		return nil, fmt.Errorf("parseing job labels: %w", err)
	}

	constraints, err := parse.NodeSelector(opts.SpecSettings.Selector)
	if err != nil {
		return nil, fmt.Errorf("parseing job contstrints: %w", err)
	}
	job := &models.Job{
		Name:        "TODO",
		Namespace:   "TODO",
		Type:        models.JobTypeBatch,
		Priority:    0,
		Count:       opts.DealSettings.Concurrency,
		Constraints: constraints,
		Labels:      labels,
		Tasks:       []*models.Task{task},
	}

	return job, nil
}

func parseWasmEntryModule(ctx context.Context, in string) (*models.SpecConfig, error) {
	// TODO(forrest) [refactor]: we need to remove this "feature" of pulling from ipfs.
	// Try interpreting this as a CID.
	wasmCid, err := cid.Parse(in)
	if err == nil {
		// It is a valid CID – proceed to create IPFS context.
		// TODO(forrest): doesn't this require a name?
		return models.NewSpecConfig(models.StorageSourceIPFS).
			WithParam("cid", wasmCid.String()), nil
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
	loader := wasm.NewModuleLoader(engine, config)
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
