package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor/wasm"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func init() { //nolint:gochecknoinits // idiomatic for cobra commands
	wasmCmd.AddCommand(runWasmCommand)
	wasmCmd.AddCommand(validateWasmCommand)
}

var wasmCmd = &cobra.Command{
	Use:   "wasm",
	Short: "Run and prepare WASM jobs on the network",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that the server version is compatible with the client version
		serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			Fatal(fmt.Sprintf("version validation failed: %s", err), 1)
			return err
		}

		return nil
	},
}

var runWasmCommand = &cobra.Command{
	Use:     "run",
	Short:   "Run a WASM job on the network",
	Long:    languageRunLong,
	Example: languageRunExample,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		ctx, rootSpan := system.NewRootSpan(cmd.Context(), system.GetTracer(), "cmd/bacalhau/wasm_run.runWasmCommand")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		programPath := args[0]
		OLR.ContextPath = programPath
		OLR.Command = args[1]

		return SubmitLanguageJob(cmd, ctx, "wasm", "2.0", programPath)
	},
}

var validateWasmCommand = &cobra.Command{
	Use:   "validate",
	Short: "Check that a WASM program is runnable on the network",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		ctx, rootSpan := system.NewRootSpan(cmd.Context(), system.GetTracer(), "cmd/bacalhau/wasm_run.validateWasmCommand")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		programPath := args[0]
		entryPoint := args[1]

		engine := wazero.NewRuntime(ctx)
		module, err := wasm.LoadModule(ctx, engine, programPath)
		if err != nil {
			Fatal(err.Error(), 1)
			return err
		}

		wasi, err := wasi_snapshot_preview1.NewBuilder(engine).Compile(ctx)
		if err != nil {
			Fatal(err.Error(), 3)
			return err
		}

		err = wasm.ValidateModuleImports(module, wasi)
		if err != nil {
			Fatal(err.Error(), 2)
			return err
		}

		err = wasm.ValidateModuleAsEntryPoint(module, entryPoint)
		if err != nil {
			Fatal(err.Error(), 2)
			return err
		}

		cmd.Println("OK")
		return nil
	},
}
