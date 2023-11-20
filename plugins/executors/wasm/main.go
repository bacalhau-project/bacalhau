package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

const PluggableExecutorPluginName = "PLUGGABLE_EXECUTOR"

// HandshakeConfig is used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "EXECUTOR_PLUGIN",
	MagicCookieValue: "bacalhau_executor",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "wasm-plugin",
		Output: os.Stderr,
		Level:  hclog.Trace,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         map[string]plugin.Plugin{
			// PluggableExecutorPluginName: &executor.ExecutorGRPCPlugin{
			// 	//Impl: wasmExecutor,
			// },
		},
		Logger:     logger,
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
