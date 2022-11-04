package bacalhau

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor/wasm"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/targzip"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const null rune = 0

var wasmJob *model.Job

var runtimeSettings *RunTimeSettings
var downloadSettings *ipfs.IPFSDownloadSettings

func init() { //nolint:gochecknoinits // idiomatic for cobra commands
	wasmJob, _ = model.NewJobWithSaneProductionDefaults()
	wasmJob.Spec.Engine = model.EngineWasm
	wasmJob.Spec.Verifier = model.VerifierDeterministic
	wasmJob.Spec.Wasm.EntryPoint = "_start"
	wasmJob.Spec.Wasm.EnvironmentVariables = map[string]string{}
	wasmJob.Spec.Outputs = []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	}

	wasmCmd.AddCommand(
		runWasmCommand,
		validateWasmCommand,
	)

	runtimeSettings = NewRunTimeSettings()
	settingsFlags := NewRunTimeSettingsFlags(runtimeSettings)
	runWasmCommand.Flags().AddFlagSet(settingsFlags)

	downloadSettings = ipfs.NewIPFSDownloadSettings()
	downloadFlags := NewIPFSDownloadFlags(downloadSettings)
	runWasmCommand.Flags().AddFlagSet(downloadFlags)

	runWasmCommand.PersistentFlags().Var(
		VerifierFlag(&wasmJob.Spec.Verifier), "verifier",
		`What verification engine to use to run the job`,
	)
	runWasmCommand.PersistentFlags().Var(
		PublisherFlag(&wasmJob.Spec.Publisher), "publisher",
		`What publisher engine to use to publish the job results`,
	)
	runWasmCommand.PersistentFlags().IntVarP(
		&wasmJob.Deal.Concurrency, "concurrency", "c", wasmJob.Deal.Concurrency,
		`How many nodes should run the job`,
	)
	runWasmCommand.PersistentFlags().IntVar(
		&wasmJob.Deal.Confidence, "confidence", wasmJob.Deal.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	wasmCmd.PersistentFlags().StringVar(
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
	Use:     "run {cid-of-wasm | <local.wasm>} [--entry-point <string>] [wasm-args ...]",
	Short:   "Run a WASM job on the network",
	Long:    languageRunLong,
	Example: languageRunExample,
	Args:    cobra.MinimumNArgs(1),
	PreRun:  applyPorcelainLogLevel,
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		ctx, rootSpan := system.NewRootSpan(cmd.Context(), system.GetTracer(), "cmd/bacalhau/wasm_run.runWasmCommand")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		var buf bytes.Buffer
		wasmCidOrPath := args[0]
		wasmJob.Spec.Wasm.Parameters = args[1:]

		// Try interpreting this as a CID.
		wasmCid, err := cid.Parse(wasmCidOrPath)
		if err == nil {
			// It is a valid CID – proceed to create IPFS context.
			wasmJob.Spec.Contexts = append(wasmJob.Spec.Contexts, model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				CID:           wasmCid.String(),
				Path:          "/job",
			})
		} else {
			// Try interpreting this as a path.
			info, err := os.Stat(wasmCidOrPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%q is not a valid CID or local file: %s", wasmCidOrPath, err.Error())
				} else {
					return err
				}
			}

			if !info.Mode().IsRegular() {
				return fmt.Errorf("%s should point to a single file", wasmCidOrPath)
			}

			err = os.Chdir(filepath.Dir(wasmCidOrPath))
			if err != nil {
				return err
			}

			cmd.Printf("Uploading %q to server to execute command in context, press Ctrl+C to cancel\n", wasmCidOrPath)
			time.Sleep(1 * time.Second)
			err = targzip.Compress(ctx, filepath.Base(wasmCidOrPath), &buf)
			if err != nil {
				return err
			}
		}

		// We can only use a Deterministic verifier if we have multiple nodes running the job
		// If the user has selected a Deterministic verifier (or we are using it by default)
		// then switch back to a Noop Verifier if the concurrency is too low.
		if wasmJob.Deal.Concurrency <= 1 && wasmJob.Spec.Verifier == model.VerifierDeterministic {
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

		return ExecuteJob(ctx, cm, cmd, wasmJob, *runtimeSettings, *downloadSettings, &buf)
	},
}

var validateWasmCommand = &cobra.Command{
	Use:   "validate <local.wasm> [--entry-point <string>]",
	Short: "Check that a WASM program is runnable on the network",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		ctx, rootSpan := system.NewRootSpan(cmd.Context(), system.GetTracer(), "cmd/bacalhau/wasm_run.validateWasmCommand")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		programPath := args[0]
		entryPoint := wasmJob.Spec.Wasm.EntryPoint

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
