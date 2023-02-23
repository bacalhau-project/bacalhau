package bidstrategy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/logger"
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
		return NewShouldBidResponse(), nil
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
		"PATH=" + os.Getenv("PATH"),
	}
	cmd.Stdin = strings.NewReader(string(jsonData))
	buf := bytes.Buffer{}
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		// we ignore this error because it might be the script exiting 1 on purpose
		logger.LogStream(ctx, &buf)
		log.Ctx(ctx).Debug().Err(err).Str("Command", s.command).Msg("We got an error back from a job selection probe exec")
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode == 0 {
		return NewShouldBidResponse(), nil
	}
	return BidStrategyResponse{
		ShouldBid: false,
		Reason:    fmt.Sprintf("command `%s` returned non-zero exit code %d", s.command, exitCode),
	}, nil
}

func (s *ExternalCommandStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return NewShouldBidResponse(), nil
}
