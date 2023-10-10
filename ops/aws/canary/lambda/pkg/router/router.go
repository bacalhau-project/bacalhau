package router

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/scenarios"
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

func init() {
	// init system configs and repo.
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get home dir: %s", err)
		os.Exit(1)
	}
	if _, err := setup.SetupBacalhauRepo(filepath.Join(home, ".bacalhau_canary")); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize bacalhau repo: %s", err)
		os.Exit(1)
	}
}

func Route(ctx context.Context, event models.Event) error {
	handler, ok := TestcasesMap[event.Action]
	if !ok {
		return fmt.Errorf("no handler found for action: %s", event.Action)
	}
	err := handler(ctx)
	if err != nil {
		return fmt.Errorf("testcase %s failed: %s", event.Action, err)
	}
	log.Info().Msgf("testcase %s passed", event.Action)
	return nil
}
