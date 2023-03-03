package bidstrategy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ExternalHTTPStrategyParams struct {
	URL string
}

type ExternalHTTPStrategy struct {
	url string
}

func NewExternalHTTPStrategy(params ExternalHTTPStrategyParams) *ExternalHTTPStrategy {
	return &ExternalHTTPStrategy{
		url: params.URL,
	}
}

func (s *ExternalHTTPStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	if s.url == "" {
		return NewShouldBidResponse(), nil
	}

	data := getJobSelectionPolicyProbeData(request)
	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		return BidStrategyResponse{}, fmt.Errorf("ExternalHTTPStrategy: error marshaling job selection policy probe data: %w", err)
	}

	body := bytes.NewBuffer(jsonData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, body)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		log.Ctx(ctx).Error().Msgf("could not create http request with context: %s", s.url)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	if err != nil {
		return BidStrategyResponse{},
			fmt.Errorf("ExternalHTTPStrategy: error http POST job selection policy probe data: %s %w", s.url, err)
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, s.url, resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("url `%s` returned %d status code", s.url, resp.StatusCode),
		}, nil
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return NewShouldBidResponse(), nil
	}

	if resp.ContentLength > int64(model.MaxSerializedStringInput) {
		return BidStrategyResponse{},
			fmt.Errorf("http result too large (%d > %d)", resp.ContentLength, model.MaxSerializedStringInput)
	}

	buf := make([]byte, resp.ContentLength)
	read, err := resp.Body.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return BidStrategyResponse{}, errors.Wrap(err, "error reading http response")
	} else if int64(read) < resp.ContentLength {
		return BidStrategyResponse{}, fmt.Errorf("only read %d, expecting %d", read, resp.ContentLength)
	}

	var result BidStrategyResponse
	err = model.JSONUnmarshalWithMax(buf, &result)
	if err != nil {
		return BidStrategyResponse{}, errors.Wrap(err, "error unmarshalling http response")
	}

	return result, nil
}

func (s *ExternalHTTPStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return NewShouldBidResponse(), nil
}
