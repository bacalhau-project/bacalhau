package supervisor

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/environment"
	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/registry"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/hashicorp/go-multierror"
)

// Supervisor is responsible for supervising the running of an Execution within
// a specific Environment.
type Supervisor struct {
	ctx      context.Context
	registry *registry.Registry

	muPlugins sync.Mutex
	plugins   map[string]*PluginProcess

	muTasks sync.Mutex
	tasks   map[*models.Execution]*environment.Environment
}

// New creates a new supervisor given a registry full of
// plugin configuration.
func New(ctx context.Context, reg *registry.Registry) *Supervisor {
	return &Supervisor{
		ctx:      ctx,
		registry: reg,
		plugins:  make(map[string]*PluginProcess),
		tasks:    make(map[*models.Execution]*environment.Environment),
	}
}

// InitialisePlugins iterates through the registry and launches the relevant
// plugins
func (s *Supervisor) InitialisePlugins() error {
	return s.registry.ForEachEntry(func(key string, cfg *registry.Config) error {
		pp := NewPluginProcess(cfg)
		if err := pp.Start(); err != nil {
			return err
		}

		s.plugins[key] = pp

		return nil
	})
}

// StopPlugins attempts to shut down all of the currently running plugins
func (s *Supervisor) StopPlugins() error {
	// Get a lock on the plugins mutex, and keep it for the entire lifetime
	// of this function to avoid other calls from being processed while we
	// shutdown.
	s.muPlugins.Lock()
	defer s.muPlugins.Unlock()

	var errors *multierror.Error

	// Stop any running executions
	s.muTasks.Lock()
	for execution := range s.tasks {
		if err := s.StopExecution(execution); err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	s.muTasks.Unlock()

	// Stop the plugins executing the code
	for _, plugin := range s.plugins {
		plugin.Stop()
	}

	// Shutdown the plugin process.
	return errors.ErrorOrNil()
}

func (s *Supervisor) LaunchExecution(execution *models.Execution, environment *environment.Environment) error {
	stask := &SupervisedTask{
		engine:      execution.Job.Task().Engine.Type,
		execution:   execution,
		environment: environment,
	}

	s.muPlugins.Lock()
	pluginProc := s.plugins[stask.engine]
	s.muPlugins.Unlock()

	// Get the client and call Start once we have populated the relevant type
	pluginProc.Client()
	// TODO:

	// if we've successfully made the call, then we can safely store the supervised task
	// to respond to other calls.
	s.muTasks.Lock()
	s.tasks[stask.execution] = environment
	s.muTasks.Unlock()

	return nil
}

func (s *Supervisor) GetExecutionStatus(execution *models.Execution) error {
	s.muTasks.Lock()
	_, found := s.tasks[execution]
	s.muTasks.Unlock()

	if !found {
		return ErrExecutionNotSupervised(execution.ID)
	}

	s.muPlugins.Lock()
	pluginProc := s.plugins[execution.Job.Task().Engine.Type]
	s.muPlugins.Unlock()

	pluginProc.Client()
	// TODO: Call Status and return the data

	return nil
}

func (s *Supervisor) StopExecution(execution *models.Execution) error {
	s.muTasks.Lock()
	env, found := s.tasks[execution]
	delete(s.tasks, execution)
	s.muTasks.Unlock()

	if !found {
		return ErrExecutionNotSupervised(execution.ID)
	}

	// Make sure the process running the plugin has stopped the
	// execution. We will do this by cancelling the `supervise`
	// goroutine which should be blocking on the related context.
	s.cleanup(env)

	return nil
}

func (s *Supervisor) cleanup(env *environment.Environment) error {
	// Remove domain sockets used for the RPC connections

	// Ask the environment to remove any resources/assets it has
	// inline with any policies it implements around delaying
	// deletion or pending job entire job completion.
	env.Destroy()

	return nil
}
