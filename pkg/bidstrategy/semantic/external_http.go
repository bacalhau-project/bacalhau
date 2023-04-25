package semantic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type ExternalHTTPStrategyParams struct {
	URL string
}

type ExternalHTTPStrategy struct {
	url string
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*ExternalHTTPStrategy)(nil)

func NewExternalHTTPStrategy(params ExternalHTTPStrategyParams) *ExternalHTTPStrategy {
	return &ExternalHTTPStrategy{
		url: params.URL,
	}
}

func (s *ExternalHTTPStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if s.url == "" {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	data := bidstrategy.GetJobSelectionPolicyProbeData(request)
	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		return bidstrategy.BidStrategyResponse{}, fmt.Errorf("ExternalHTTPStrategy: error marshaling job selection policy probe data: %w", err)
	}

	body := bytes.NewBuffer(jsonData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, body)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		log.Ctx(ctx).Error().Msgf("could not create http request with context: %s", s.url)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	if err != nil {
		return bidstrategy.BidStrategyResponse{},
			fmt.Errorf("ExternalHTTPStrategy: error http POST job selection policy probe data: %s %w", s.url, err)
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, s.url, resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("url `%s` returned %d status code", s.url, resp.StatusCode),
		}, nil
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	if resp.ContentLength > int64(model.MaxSerializedStringInput) {
		return bidstrategy.BidStrategyResponse{},
			fmt.Errorf("http result too large (%d > %d)", resp.ContentLength, model.MaxSerializedStringInput)
	}

	buf := make([]byte, resp.ContentLength)
	read, err := resp.Body.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return bidstrategy.BidStrategyResponse{}, errors.Wrap(err, "error reading http response")
	} else if int64(read) < resp.ContentLength {
		return bidstrategy.BidStrategyResponse{}, fmt.Errorf("only read %d, expecting %d", read, resp.ContentLength)
	}

	var result bidstrategy.BidStrategyResponse
	err = model.JSONUnmarshalWithMax(buf, &result)
	if err != nil {
		return bidstrategy.BidStrategyResponse{}, errors.Wrap(err, "error unmarshalling http response")
	}

	return result, nil
}
