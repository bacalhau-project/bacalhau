package router

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/scenarios"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

var TestcasesMap = map[string]Handler{
	"list":         scenarios.List,
	"submit":       scenarios.Submit,
	"submitAndGet": scenarios.SubmitAndGet,
	//skipping submitDockerIPFSJobAndGet as it is not stable yet: https://github.com/bacalhau-project/bacalhau/issues/1869
	//"submitDockerIPFSJobAndGet": scenarios.SubmitDockerIPFSJobAndGet,
	"submitAndDescribe":     scenarios.SubmitAnDescribe,
	"submitWithConcurrency": scenarios.SubmitWithConcurrency,
}

func init() {
	// init system configs
	err := system.InitConfig()
	if err != nil {
		panic(err)
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
