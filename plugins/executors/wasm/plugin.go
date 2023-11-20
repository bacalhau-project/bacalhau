package main

import (
	"log"

	"bacalhau-exec-wasm/executor"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type ExecutorGRPCPlugin struct {
	plugin.Plugin
	Impl WasmExecutor
}

func (p *ExecutorGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	wasmExecutor, err := NewWasmExecutor()
	if err != nil {
		log.Fatal(err)
	}

	executor.RegisterExecutorServer(s, wasmExecutor)
	return nil
}
