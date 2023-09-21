package wasm

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

	// "github.com/dylibso/observe-sdk/go"
	observe "github.com/dylibso/observe-sdk/go"
	"github.com/dylibso/observe-sdk/go/adapter/stdout"
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
	wazero.Runtime
	adapter *stdout.StdoutAdapter
}

// tracedModule wraps a 'real' wazero api.Module so that function calls made to the module can be easily traced.
type tracedModule struct {
	api.Module
	adapter  *stdout.StdoutAdapter
	traceCtx *observe.TraceCtx
}

// tracedFunction wraps a 'real' wazero api.Function so that calls to the function can be easily traced.
type tracedFunction struct {
	api.Function
	adapter  *stdout.StdoutAdapter
	traceCtx *observe.TraceCtx
}

type globalTraceContext struct {
	lock     sync.Mutex
	traceCtx *observe.TraceCtx
}

// TODO(dylibso): hack to get TraceCtx for CompiledModules, this will hold a lock when `CompileModule` is called that is released in `InstantiateModule`
// the real fix for this is probably to create a `tracedCompiledModule` wrapper type like the ones above
var lastTraceCtx globalTraceContext

func (t tracedRuntime) Instantiate(ctx context.Context, source []byte) (api.Module, error) {
	t.adapter.Start(ctx)
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.Instantiate")
	defer span.End()
	traceCtx, err := t.adapter.NewTraceCtx(ctx, t.Runtime, source, nil)
	if err != nil {
		return nil, err
	}
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.Runtime.Instantiate(ctx, source))

	if module != nil {
		module = tracedModule{Module: module, adapter: t.adapter, traceCtx: traceCtx}
	}
	return module, err
}

func (t tracedRuntime) InstantiateWithConfig(ctx context.Context, source []byte, config wazero.ModuleConfig) (api.Module, error) {
	t.adapter.Start(ctx)
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.InstantiateWithConfig")
	defer span.End()
	traceCtx, err := t.adapter.NewTraceCtx(ctx, t.Runtime, source, nil)
	if err != nil {
		return nil, err
	}
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.Runtime.InstantiateWithConfig(ctx, source, config))
	if module != nil {
		module = tracedModule{Module: module, adapter: t.adapter, traceCtx: traceCtx}
	}
	return module, err
}

func (t tracedRuntime) CompileModule(ctx context.Context, binary []byte) (wazero.CompiledModule, error) {
	t.adapter.Start(ctx)
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.CompileModule")
	defer span.End()
	traceCtx, err := t.adapter.NewTraceCtx(ctx, t.Runtime, binary, nil)
	if err != nil {
		return nil, err
	}
	module, err := telemetry.RecordErrorOnSpanTwo[wazero.CompiledModule](span)(t.Runtime.CompileModule(ctx, binary))
	if module != nil {
		if name := module.Name(); name != "" {
			span.SetAttributes(semconv.CodeNamespace(name))
		}
	}
	lastTraceCtx.lock.Lock()
	lastTraceCtx.traceCtx = traceCtx
	return module, err
}

func (t tracedRuntime) InstantiateModule(
	ctx context.Context,
	compiled wazero.CompiledModule,
	config wazero.ModuleConfig,
) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.tracedRuntime.InstantiateModule")
	defer span.End()
	module, err := telemetry.RecordErrorOnSpanTwo[api.Module](span)(t.Runtime.InstantiateModule(ctx, compiled, config))
	if err == nil && module != nil {
		if name := module.Name(); name != "" {
			span.SetAttributes(semconv.CodeNamespace(name))
		}
		module = tracedModule{Module: module, adapter: t.adapter, traceCtx: lastTraceCtx.traceCtx}
	}
	lastTraceCtx.lock.Unlock()
	return module, err
}

func (t tracedModule) ExportedFunction(name string) api.Function {
	return tracedFunction{Function: t.Module.ExportedFunction(name), adapter: t.adapter, traceCtx: t.traceCtx}
}

func (t tracedFunction) Call(ctx context.Context, params ...uint64) ([]uint64, error) {
	ctx, span := system.NewSpan(
		ctx,
		system.GetTracer(),
		"pkg/executor/wasm.tracedFunction.Call",
		trace.WithAttributes(semconv.CodeFunction(t.Function.Definition().Name())),
	)
	defer span.End()
	defer t.traceCtx.Finish()

	return telemetry.RecordErrorOnSpanTwo[[]uint64](span)(t.Function.Call(ctx, params...))
}

func (t tracedFunction) CallWithStack(ctx context.Context, stack []uint64) error {
	ctx, span := system.NewSpan(
		ctx,
		system.GetTracer(),
		"pkg/executor/wasm.tracedFunction.CallWithStack",
		trace.WithAttributes(semconv.CodeFunction(t.Function.Definition().Name())),
	)
	defer span.End()
	defer t.traceCtx.Finish()
	return telemetry.RecordErrorOnSpan(span)(t.Function.CallWithStack(ctx, stack))
}

// Functions below this line just forward straight to the delegate

func (t tracedRuntime) NewHostModuleBuilder(moduleName string) wazero.HostModuleBuilder {
	return t.Runtime.NewHostModuleBuilder(moduleName)
}

func (t tracedRuntime) CloseWithExitCode(ctx context.Context, exitCode uint32) error {
	t.adapter.Stop(true)
	return t.Runtime.CloseWithExitCode(ctx, exitCode)
}

func (t tracedRuntime) Module(moduleName string) api.Module {
	return t.Runtime.Module(moduleName)
}

func (t tracedRuntime) Close(ctx context.Context) error {
	t.adapter.Stop(true)
	return t.Runtime.Close(ctx)
}

func (t tracedFunction) Definition() api.FunctionDefinition {
	return t.Function.Definition()
}

func (t tracedModule) String() string {
	return t.Module.String()
}

func (t tracedModule) Name() string {
	return t.Module.Name()
}

func (t tracedModule) Memory() api.Memory {
	return t.Module.Memory()
}

func (t tracedModule) ExportedFunctionDefinitions() map[string]api.FunctionDefinition {
	return t.Module.ExportedFunctionDefinitions()
}

func (t tracedModule) ExportedMemory(name string) api.Memory {
	return t.Module.ExportedMemory(name)
}

func (t tracedModule) ExportedMemoryDefinitions() map[string]api.MemoryDefinition {
	return t.Module.ExportedMemoryDefinitions()
}

func (t tracedModule) ExportedGlobal(name string) api.Global {
	return t.Module.ExportedGlobal(name)
}

func (t tracedModule) CloseWithExitCode(ctx context.Context, exitCode uint32) error {
	return t.Module.CloseWithExitCode(ctx, exitCode)
}

func (t tracedModule) Close(ctx context.Context) error {
	// t.traceCtx.Finish()
	return t.Module.Close(ctx)
}
