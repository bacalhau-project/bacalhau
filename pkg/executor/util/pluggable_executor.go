package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-plugin"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/plugins/grpc"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewPluggableExecutor() *PluggableExecutor {
	return &PluggableExecutor{
		registered: make(map[string]PluginExecutorConfig),
		active:     make(map[string]*activeExecutor),
	}
}

type PluggableExecutor struct {
	registered map[string]PluginExecutorConfig
	active     map[string]*activeExecutor
}

func (e *PluggableExecutor) Get(ctx context.Context, key model.Engine) (executor.Executor, error) {
	engine, ok := e.active[key.String()]
	if !ok {
		return nil, fmt.Errorf("pluging %s not found", key)
	}
	return engine.Impl, nil
}

func (e *PluggableExecutor) Has(ctx context.Context, key model.Engine) bool {
	_, ok := e.active[key.String()]
	return ok
}

type activeExecutor struct {
	Impl   executor.Executor
	Closer func()
}

type PluginExecutorConfig struct {
	Name             string
	Path             string
	Command          string
	ProtocolVersion  uint
	MagicCookieKey   string
	MagicCookieValue string
}

func (e *PluggableExecutor) RegisterPlugin(config PluginExecutorConfig) error {
	_, ok := e.registered[config.Name]
	if ok {
		return fmt.Errorf("duplicate registration of exector %s", config.Name)
	}

	if pluginBin, err := os.Stat(filepath.Join(config.Path, config.Name)); err != nil {
		return err
	} else if pluginBin.IsDir() {
		return fmt.Errorf("plugin location is directory, expected binary")
	}
	// TODO check if binary is executable

	e.registered[config.Name] = config
	return nil
}

func (e *PluggableExecutor) Start(ctx context.Context) error {
	for name, config := range e.registered {
		pluginExecutor, closer, err := e.dispense(name, config)
		if err != nil {
			return err
		}
		e.active[name] = &activeExecutor{
			Impl:   pluginExecutor,
			Closer: closer,
		}
	}
	return nil
}

func (e *PluggableExecutor) Stop(ctx context.Context) error {
	for _, active := range e.active {
		active.Closer()
	}
	return nil
}

const PluggableExecutorPluginName = "PLUGGABLE_EXECUTOR"

func (e *PluggableExecutor) dispense(name string, config PluginExecutorConfig) (executor.Executor, func(), error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		Plugins: map[string]plugin.Plugin{
			PluggableExecutorPluginName: &grpc.ExecutorGRPCPlugin{},
		},
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  config.ProtocolVersion,
			MagicCookieKey:   config.MagicCookieKey,
			MagicCookieValue: config.MagicCookieValue,
		},
		Cmd: exec.Command(filepath.Join(config.Path, config.Command)),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, nil, err
	}

	raw, err := rpcClient.Dispense(PluggableExecutorPluginName)
	if err != nil {
		client.Kill()
		return nil, nil, err
	}

	pluginExecutor, ok := raw.(executor.Executor)
	if !ok {
		client.Kill()
		return nil, nil, fmt.Errorf("plugin is not of type: PluggableExecutor")
	}

	return pluginExecutor, func() { client.Kill() }, nil
}
