package router

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/scenarios"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

var TestcasesMap = map[string]Handler{
	"list":                            scenarios.List,
	"submit":                          scenarios.Submit,
	"submitAndGet":                    scenarios.SubmitAndGet,
	"submitDockerIPFSJobAndGet":       scenarios.SubmitDockerIPFSJobAndGet,
	"submitAndDescribe":               scenarios.SubmitAnDescribe,
	"submitWithConcurrency":           scenarios.SubmitWithConcurrency,
	"submitWithConcurrencyOwnedNodes": scenarios.SubmitWithConcurrencyOwnedNodes,
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
