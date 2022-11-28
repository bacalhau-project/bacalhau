package bidstrategy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type ExternalCommandStrategyParams struct {
	Command string
}

type ExternalCommandStrategy struct {
	command string
}

func NewExternalCommandStrategy(params ExternalCommandStrategyParams) *ExternalCommandStrategy {
	return &ExternalCommandStrategy{
		command: params.Command,
	}
}

func (s *ExternalCommandStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	if s.command == "" {
		return newShouldBidResponse(), nil
	}

	// TODO: Use context to trace exec call

	data := getJobSelectionPolicyProbeData(request)
	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		return BidStrategyResponse{},
			fmt.Errorf("ExternalCommandStrategy: error marshaling job selection policy probe data: %w", err)
	}

	cmd := exec.Command("bash", "-c", s.command) //nolint:gosec
	cmd.Env = []string{
		"BACALHAU_JOB_SELECTION_PROBE_DATA=" + string(jsonData),
	}
	cmd.Stdin = strings.NewReader(string(jsonData))
	err = cmd.Run()
	if err != nil {
		// we ignore this error because it might be the script exiting 1 on purpose
		log.Ctx(ctx).Debug().Msgf("We got an error back from a job selection probe exec: %s %s", s.command, err.Error())
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode == 0 {
		return newShouldBidResponse(), nil
	}
	return BidStrategyResponse{
		ShouldBid: false,
		Reason:    fmt.Sprintf("command `%s` returned non-zero exit code %d", s.command, exitCode),
	}, nil
}

func (s *ExternalCommandStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}
