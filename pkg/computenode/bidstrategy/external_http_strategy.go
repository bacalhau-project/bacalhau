package bidstrategy

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type ExternalHttpStrategyParams struct {
	Url string
}

type ExternalHttpStrategy struct {
	url string
}

func NewExternalHttpStrategy(params ExternalHttpStrategyParams) *ExternalHttpStrategy {
	return &ExternalHttpStrategy{
		url: params.Url,
	}
}

func (s *ExternalHttpStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	if s.url == "" {
		return newShouldBidResponse(), nil
	}

	data := getJobSelectionPolicyProbeData(request)
	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		return BidStrategyResponse{}, fmt.Errorf("ExternalHttpStrategy: error marshaling job selection policy probe data: %w", err)
	}

	body := bytes.NewBuffer(jsonData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, body)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		log.Ctx(ctx).Error().Msgf("could not create http request with context: %s", s.url)
	}
	resp, err := http.DefaultClient.Do(req)
	resp.Body.Close()

	if err != nil {
		return BidStrategyResponse{},
			fmt.Errorf("ExternalHttpStrategy: error http POST job selection policy probe data: %s %w", s.url, err)
	}

	if resp.StatusCode == http.StatusOK {
		return newShouldBidResponse(), nil
	}
	return BidStrategyResponse{
		ShouldBid: false,
		Reason:    fmt.Sprintf("url `%s` returned %d status code", s.url, resp.StatusCode),
	}, nil
}
