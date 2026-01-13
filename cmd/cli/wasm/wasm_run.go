package wasm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/cli/helpers"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/opts"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	engine_wasm "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
)

var (
	wasmRunLong = templates.LongDesc(`
		Runs a job that was compiled to WASM.

		The entry module can be specified in three ways:
		1. Local file path (e.g., ./main.wasm)
		2. Storage spec (e.g., s3://bucket/main.wasm, http://example.com/main.wasm)
		3. Target path (when the module is added via --input flag)

		You can override the target path for any of these using the path:target syntax:
		- Local file: ./main.wasm:/app/custom.wasm
		- Storage spec: s3://bucket/main.wasm:/app/custom.wasm

		Import modules must be added via the --input flag and referenced by their target paths.
		`)

	wasmRunExample = templates.Examples(`
		# Run a WASM module from local file
		bacalhau wasm run ./main.wasm

		# Run a WASM module from S3
		bacalhau wasm run s3://bucket/main.wasm

		# Run a WASM module from HTTP
		bacalhau wasm run http://example.com/main.wasm

		# Run a WASM module with custom target path
		bacalhau wasm run ./main.wasm:/app/custom.wasm
		bacalhau wasm run s3://bucket/main.wasm:/app/custom.wasm

		# Run a WASM module with import modules
		bacalhau wasm run --input s3://bucket/lib.wasm:/app/lib.wasm s3://bucket/main.wasm --import-modules /app/lib.wasm
		`)
)

type WasmRunOptions struct {
	// Target paths for import modules that will be available to the entry module
	ImportModules []string
	// The name of the WASM function to call in the entry module
	Entrypoint string

	JobSettings     *cliflags.JobSettings
	TaskSettings    *cliflags.TaskSettings
	RunTimeSettings *cliflags.RunTimeSettings
}

func NewWasmOptions() *WasmRunOptions {
	return &WasmRunOptions{
		ImportModules: []string{},
		Entrypoint:    "_start",

		JobSettings:     cliflags.DefaultJobSettings(),
		TaskSettings:    cliflags.DefaultTaskSettings(),
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
	}
}

func NewCmd() *cobra.Command {
	wasmCmd := &cobra.Command{
		Use:   "wasm",
		Short: "Run and prepare WASM jobs on the network",
	}

	wasmCmd.AddCommand(newRunCmd())
	return wasmCmd
}

func newRunCmd() *cobra.Command {
	opts := NewWasmOptions()

	wasmRunCmd := &cobra.Command{
		Use:      "run ENTRY-MODULE [wasm-args ...]",
		Short:    "Run a WASM job on the network",
		Long:     wasmRunLong,
		Example:  wasmRunExample,
		Args:     cobra.MinimumNArgs(1),
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			return run(cmd, cmdArgs, cfg, opts)
		},
	}

	cliflags.RegisterJobFlags(wasmRunCmd, opts.JobSettings)
	cliflags.RegisterTaskFlags(wasmRunCmd, opts.TaskSettings)
	wasmRunCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(opts.RunTimeSettings))

	// register flags unique to wasm.
	wasmFlags := pflag.NewFlagSet("wasm", pflag.ContinueOnError)
	wasmFlags.StringSliceVarP(&opts.ImportModules, "import-modules", "I", []string{},
		`Target paths of WASM modules to import. These paths must match the target paths specified in the --input flags.
		For example, if you added a module with --input s3://bucket/lib.wasm:/app/lib.wasm, use --import-modules /app/lib.wasm`,
	)
	wasmFlags.StringVar(&opts.Entrypoint, "entry-point", opts.Entrypoint,
		`The name of the WASM function in the entry module to call. This should be a zero-parameter zero-result function that
		will execute the job.`,
	)

	wasmRunCmd.Flags().AddFlagSet(wasmFlags)
	return wasmRunCmd
}

func run(cmd *cobra.Command, args []string, cfg types.Bacalhau, opts *WasmRunOptions) error {
	ctx := cmd.Context()

	job, err := build(ctx, args, opts)
	if err != nil {
		return err
	}

	if opts.RunTimeSettings.DryRun {
		out, err := helpers.JobToYaml(job)
		if err != nil {
			return err
		}
		cmd.Print(out)
		return nil
	}

	// Only create API client when actually needed (not for dry-run)
	api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create api client: %w", err)
	}

	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	if !opts.RunTimeSettings.PrintJobIDOnly && len(resp.Warnings) > 0 {
		printer.PrintWarnings(cmd, resp.Warnings)
		cmd.Println()
	}

	job.ID = resp.JobID
	jobProgressPrinter := printer.NewJobProgressPrinter(api, opts.RunTimeSettings)
	if err := jobProgressPrinter.PrintJobProgress(ctx, job, cmd); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

// parseWasmModule handles the entry module path and returns an InputSource if it's a local file or storage spec.
// If it's a target path, it returns nil and the path should be treated as-is.
func parseWasmModule(ctx context.Context, in string, defaultTarget string) (*models.InputSource, error) {
	// Try interpreting this as a storage spec (http://, s3://, etc.)
	spec, err := opts.ParseStorageSpec(in, defaultTarget)
	if err == nil {
		return spec, nil
	}

	// Try interpreting this as a local path with target override
	var filePath, targetPath string
	if parts := strings.Split(in, ":"); len(parts) == 2 {
		filePath = parts[0]
		targetPath = parts[1]
	} else {
		filePath = in
		targetPath = defaultTarget
	}

	// Check if it's a local file
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Not a local file, treat as target path
			return nil, nil
		}
		return nil, err
	}

	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q should point to a single file", filePath)
	}

	storage := inline.NewStorage()
	inlineData, err := storage.Upload(ctx, filePath)
	if err != nil {
		return nil, err
	}

	return &models.InputSource{
		Source: &inlineData,
		Target: targetPath,
	}, nil
}

func build(ctx context.Context, args []string, opts *WasmRunOptions) (*models.Job, error) {
	entryModule := args[0]

	// Try to handle the entry module as a local file or storage spec
	inputSource, err := parseWasmModule(ctx, entryModule, "main.wasm")
	if err != nil {
		return nil, fmt.Errorf("failed to parse entry module: %w", err)
	}

	// If it's a local file or storage spec, add it to the inputs
	if inputSource != nil {
		opts.TaskSettings.InputSources.AddValue(inputSource)
		// Use the target path as the entry module path
		entryModule = inputSource.Target
	}

	// Process import modules
	var importModulePaths []string
	for _, importModule := range opts.ImportModules {
		// Try to handle the import module as a local file or storage spec
		moduleInputSource, err := parseWasmModule(ctx, importModule, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse import module %q: %w", importModule, err)
		}

		// If it's a local file or storage spec, add it to the inputs
		if moduleInputSource != nil {
			if moduleInputSource.Target == "" {
				return nil, fmt.Errorf("import module %q must specify a target path using the format 'path:target'", importModule)
			}
			opts.TaskSettings.InputSources.AddValue(moduleInputSource)
			// Use the target path as the import module path
			importModulePaths = append(importModulePaths, moduleInputSource.Target)
		} else {
			// If it's just a target path, use it as-is
			importModulePaths = append(importModulePaths, importModule)
		}
	}

	// Build engine spec using the target paths
	engineSpec, err := engine_wasm.NewWasmEngineBuilder(entryModule).
		WithParameters(args[1:]...).
		WithEntrypoint(opts.Entrypoint).
		WithImportModules(importModulePaths).
		Build()
	if err != nil {
		return nil, err
	}

	return helpers.BuildJobFromFlags(engineSpec, opts.JobSettings, opts.TaskSettings)
}
