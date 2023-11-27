package supervisor

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/registry"
)

type PluginProcess struct {
	ctx     context.Context
	cancel  context.CancelFunc
	config  *registry.Config
	command *exec.Cmd

	serviceSocket    string
	superVisorSocket string

	// client
}

func NewPluginProcess(config *registry.Config) *PluginProcess {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	serviceSkt := ""
	supervisorSkt := ""

	return &PluginProcess{
		ctx:              ctx,
		cancel:           cancel,
		config:           config,
		serviceSocket:    serviceSkt,
		superVisorSocket: supervisorSkt,
	}
}

func (p *PluginProcess) Start() error {
	// Make sure there is _some_ env set for the plugin so that it doesn't inherit the
	// environment of this process. And make sure we only provide safe (non-bacalhau)
	// vars so that the plugin only has access to what we want it to see.
	var envVars []string
	envVars = append(envVars, p.config.SafeEnvironmentVariables()...)
	envVars = append(envVars, []string{
		fmt.Sprintf("BACALHAU_EXECUTOR_SOCKET=%s", p.serviceSocket),
		fmt.Sprintf("BACALHAU_SUPERVISOR_SOCKET=%s", p.superVisorSocket),
	}...)

	p.command = exec.CommandContext(p.ctx, p.config.Executable, p.config.Arguments...)
	p.command.Env = envVars

	if err := p.command.Start(); err != nil {
		return err
	}

	go p.run()
	return nil
}

func (p *PluginProcess) Client() {

}

func (p *PluginProcess) run() {
	_ = p.command.Wait()
}

func (p *PluginProcess) Stop() {
	p.cancel()
}
