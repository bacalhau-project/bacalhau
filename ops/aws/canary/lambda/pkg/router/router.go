package router

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/scenarios"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

var TestcasesMap = map[string]Handler{
	"list":                      scenarios.List,
	"submit":                    scenarios.Submit,
	"submitAndGet":              scenarios.SubmitAndGet,
	"submitDockerIPFSJobAndGet": scenarios.SubmitDockerIPFSJobAndGet,
	"submitAndDescribe":         scenarios.SubmitAnDescribe,
	"submitWithConcurrency":     scenarios.SubmitWithConcurrency,
}

type Settings struct {
	cfg *types.BacalhauConfig
}

type Option func(s *Settings)

func WithConfig(cfg types.BacalhauConfig) Option {
	return func(s *Settings) {
		s.cfg = &cfg
	}
}

func Route(ctx context.Context, event models.Event, opts ...Option) error {
	// NB(forrest): settings is required to allow this method to be called with or without config.
	// The lambda runs expect this method to instantiate a repo and config then run. No config is provided.
	// Conversely, TestScenariosAgainstDevstack creates a devstack and needs to provide
	// this method with explicit config related to the nodes in the devstack. A config is provided.
	settings := &Settings{cfg: nil}
	for _, opt := range opts {
		opt(settings)
	}
	if settings.cfg == nil {
		repoPath, err := os.MkdirTemp("", "bacalhau_canary_repo_*")
		if err != nil {
			return fmt.Errorf("failed to create repo dir: %s", err)
		}

		c := config.New()
		// init system configs and repo.
		if _, err := setup.SetupBacalhauRepo(repoPath, c); err != nil {
			return fmt.Errorf("failed to initialize bacalhau repo: %s", err)
		}

		resolvedCfg, err := c.Current()
		if err != nil {
			return err
		}
		settings.cfg = &resolvedCfg
	}

	handler, ok := TestcasesMap[event.Action]
	if !ok {
		return fmt.Errorf("no handler found for action: %s", event.Action)
	}
	if err := handler(ctx, *settings.cfg); err != nil {
		return fmt.Errorf("testcase %s failed: %s", event.Action, err)
	}
	log.Info().Msgf("testcase %s passed", event.Action)
	return nil
}
