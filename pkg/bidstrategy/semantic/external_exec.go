package semantic

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

type ExternalCommandStrategyParams struct {
	Command string
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*ExternalCommandStrategy)(nil)

type ExternalCommandStrategy struct {
	command string
}

func NewExternalCommandStrategy(params ExternalCommandStrategyParams) *ExternalCommandStrategy {
	return &ExternalCommandStrategy{
		command: params.Command,
	}
}

const (
	exitCodeReason = "accept jobs where external command %q returns exit code %d"
)

func (s *ExternalCommandStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if s.command == "" {
		return bidstrategy.NewBidResponse(true, notConfiguredReason), nil
	}

	// TODO: Use context to trace exec call

	data := bidstrategy.GetJobSelectionPolicyProbeData(request)
	jsonData, err := marshaller.JSONMarshalWithMax(data)

	if err != nil {
		return bidstrategy.BidStrategyResponse{},
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
	return bidstrategy.NewBidResponse(exitCode == 0, exitCodeReason, exitCode), nil
}
