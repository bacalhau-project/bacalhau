package wasm

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var _ wazero.Runtime = tracedRuntime{}
var _ api.Function = tracedFunction{}
var _ api.Module = tracedModule{}

// tracedRuntime wraps a 'real' wazero.Runtime so that important events like compiling modules can be easily traced.
type tracedRuntime struct {
	delegate wazero.Runtime
}

// tracedModule wraps a 'real' wazero api.Module so that function calls made to the module can be easily traced.
type tracedModule struct {
	delegate api.Module
}

// tracedFunction wraps a 'real' wazero api.Function so that calls to the function can be easily traced.
type tracedFunction struct {
	delegate api.Function
}

func (t tracedRuntime) Instantiate(ctx context.Context, source []byte) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.Instantiate")
	defer span.End()
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.delegate.Instantiate(ctx, source))
	if module != nil {
		module = tracedModule{module}
	}
	return module, err
}

func (t tracedRuntime) InstantiateWithConfig(ctx context.Context, source []byte, config wazero.ModuleConfig) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.InstantiateWithConfig")
	defer span.End()
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.delegate.InstantiateWithConfig(ctx, source, config))
	if module != nil {
		module = tracedModule{module}
	}
	return module, err
}

func (t tracedRuntime) CompileModule(ctx context.Context, binary []byte) (wazero.CompiledModule, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.CompileModule")
	defer span.End()
	module, err := telemetry.RecordErrorOnSpanTwo[wazero.CompiledModule](span)(t.delegate.CompileModule(ctx, binary))
	if module != nil {
		if name := module.Name(); name != "" {
			span.SetAttributes(semconv.CodeNamespace(name))
		}
	}
	return module, err
}

func (t tracedRuntime) InstantiateModule(
	ctx context.Context,
	compiled wazero.CompiledModule,
	config wazero.ModuleConfig,
) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.InstantiateModule")
	defer span.End()
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.delegate.InstantiateModule(ctx, compiled, config))
	if module != nil {
		if name := module.Name(); name != "" {
			span.SetAttributes(semconv.CodeNamespace(name))
		}
		module = tracedModule{module}
	}
	return module, err
}

func (t tracedModule) ExportedFunction(name string) api.Function {
	return tracedFunction{t.delegate.ExportedFunction(name)}
}

func (t tracedFunction) Call(ctx context.Context, params ...uint64) ([]uint64, error) {
	ctx, span := system.NewSpan(
		ctx,
		system.GetTracer(),
		"pkg/executor/wasm.tracedFunction.Call",
		trace.WithAttributes(semconv.CodeFunction(t.delegate.Definition().Name())),
	)
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[[]uint64](span)(t.delegate.Call(ctx, params...))
}

// Functions below this line just forward straight to the delegate

func (t tracedRuntime) NewHostModuleBuilder(moduleName string) wazero.HostModuleBuilder {
	return t.delegate.NewHostModuleBuilder(moduleName)
}

func (t tracedRuntime) CloseWithExitCode(ctx context.Context, exitCode uint32) error {
	return t.delegate.CloseWithExitCode(ctx, exitCode)
}

func (t tracedRuntime) Module(moduleName string) api.Module {
	return t.delegate.Module(moduleName)
}

func (t tracedRuntime) Close(ctx context.Context) error {
	return t.delegate.Close(ctx)
}

func (t tracedFunction) Definition() api.FunctionDefinition {
	return t.delegate.Definition()
}

func (t tracedModule) String() string {
	return t.delegate.String()
}

func (t tracedModule) Name() string {
	return t.delegate.Name()
}

func (t tracedModule) Memory() api.Memory {
	return t.delegate.Memory()
}

func (t tracedModule) ExportedFunctionDefinitions() map[string]api.FunctionDefinition {
	return t.delegate.ExportedFunctionDefinitions()
}

func (t tracedModule) ExportedMemory(name string) api.Memory {
	return t.delegate.ExportedMemory(name)
}

func (t tracedModule) ExportedMemoryDefinitions() map[string]api.MemoryDefinition {
	return t.delegate.ExportedMemoryDefinitions()
}

func (t tracedModule) ExportedGlobal(name string) api.Global {
	return t.delegate.ExportedGlobal(name)
}

func (t tracedModule) CloseWithExitCode(ctx context.Context, exitCode uint32) error {
	return t.delegate.CloseWithExitCode(ctx, exitCode)
}

func (t tracedModule) Close(ctx context.Context) error {
	return t.delegate.Close(ctx)
}
