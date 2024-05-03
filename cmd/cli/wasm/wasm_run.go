package wasm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/cli/helpers"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
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
	ImportModules        []*models.InputSource
	Entrypoint           string
	EnvironmentVariables []string

	JobSettings      *cliflags.JobSettings
	TaskSettings     *cliflags.TaskSettings
	RunTimeSettings  *cliflags.RunTimeSettings
	DownloadSettings *cliflags.DownloaderSettings
}

func NewWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		ImportModules:        make([]*models.InputSource, 0),
		Entrypoint:           "_start",
		EnvironmentVariables: []string{},
		JobSettings:          cliflags.DefaultJobSettings(),
		TaskSettings:         cliflags.DefaultTaskSettings(),
		DownloadSettings:     cliflags.DefaultDownloaderSettings(),
		RunTimeSettings:      cliflags.DefaultRunTimeSettings(),
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

	// flags with a corresponding config value via env vars, config file
	wasmRunFlags := map[string][]configflags.Definition{
		"ipfs": configflags.IPFSFlags,
	}

	cmd := &cobra.Command{
		Use:      "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
		Short:    "Run a WASM job on the network",
		Long:     wasmRunLong,
		Example:  wasmRunExample,
		Args:     cobra.MinimumNArgs(1),
		PreRunE:  hook.Chain(hook.ClientPreRunHooks, configflags.PreRun(wasmRunFlags)),
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, opts)
		},
	}

	// register config-based flags.
	if err := configflags.RegisterFlags(cmd, wasmRunFlags); err != nil {
		util.Fatal(cmd, err, 1)
	}

	// register common flags.
	cliflags.RegisterJobFlags(cmd, opts.JobSettings)
	cliflags.RegisterTaskFlags(cmd, opts.TaskSettings)
	cliflags.RegisterDownloadFlags(cmd, opts.DownloadSettings)
	cliflags.RegisterRunTimeFlags(cmd, opts.RunTimeSettings)

	// register flags unique to wasmt.
	wasmFlags := pflag.NewFlagSet("wasm", pflag.ContinueOnError)
	wasmFlags.VarP(flags.NewURLStorageSpecArrayFlag(&opts.ImportModules), "import-module-urls", "U",
		`URL of the WASM modules to import from a URL source. URL accept any valid URL supported by `+
			`the 'wget' command, and supports both HTTP and HTTPS.`,
	)
	wasmFlags.StringVar(&opts.Entrypoint, "entry-point", opts.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)
	wasmFlags.StringSliceVarP(&opts.EnvironmentVariables, "env", "e", opts.EnvironmentVariables,
		"The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)")

	return cmd
}

func run(cmd *cobra.Command, args []string, opts *WasmRunOptions) error {
	ctx := cmd.Context()

	job, err := build(ctx, args, opts)
	if err != nil {
		return err
	}

	/*
		if opts.RunTimeSettings.DryRun {
			out, err := helpers.JobToYaml(job)
			if err != nil {
				return err
			}
			cmd.Print(out)
			return nil
		}

	*/

	api := util.GetAPIClientV2(cmd)
	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	if len(resp.Warnings) > 0 {
		helpers.PrintWarnings(cmd, resp.Warnings)
	}

	/*
		if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, opts.RunTimeSettings, api); err != nil {
			return fmt.Errorf("failed to print job execution: %w", err)
		}

	*/

	return nil
}

func build(ctx context.Context, args []string, opts *WasmRunOptions) (*models.Job, error) {
	parameters := args[1:]
	entryModuleStr := args[0]

	entryModule, err := parseWasmEntryModule(ctx, entryModuleStr)
	if err != nil {
		return nil, fmt.Errorf("reading entry module: %w", err)
	}

	envvar, err := parse.StringSliceToMap(opts.EnvironmentVariables)
	if err != nil {
		return nil, fmt.Errorf("parseing environment variables: %w", err)
	}
	engineSpec, err := models.WasmSpecBuilder(&models.InputSource{
		Source: entryModule,
		Alias:  "TODO",
		Target: "TODO",
	}).WithEntrypoint(opts.Entrypoint).
		WithParameters(parameters...).
		WithEnvironmentVariables(envvar).
		WithImportModules(opts.ImportModules...).
		Build()
	if err != nil {
		return nil, fmt.Errorf("building wasm engine spec: %w", err)
	}

	job, err := helpers.BuildJobFromFlags(engineSpec, opts.JobSettings, opts.TaskSettings)
	if err != nil {
		return nil, fmt.Errorf("building job spec: %w", err)
	}

	// Normalize and validate the job spec
	job.Normalize()
	if err := job.ValidateSubmission(); err != nil {
		return nil, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
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
