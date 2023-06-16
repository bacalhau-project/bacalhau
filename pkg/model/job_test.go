//go:build unit || !integration

package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var validDeals = []Deal{
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: 0, MinBids: 0},
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: 0, MinBids: 3},
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: 1, MinBids: 0},
	{TargetingMode: TargetAny, Concurrency: 5, Confidence: 0, MinBids: 0},
	{TargetingMode: TargetAll, Concurrency: 0, Confidence: 0, MinBids: 0},
	{TargetingMode: TargetAll, Concurrency: 1, Confidence: 0, MinBids: 0},
}

var invalidDeals = []Deal{
	{},
	{TargetingMode: TargetAll, Concurrency: 2, Confidence: 0, MinBids: 0},
	{TargetingMode: TargetAll, Concurrency: 0, Confidence: 1, MinBids: 0},
	{TargetingMode: TargetAll, Concurrency: 0, Confidence: 0, MinBids: 1},
	{TargetingMode: TargetAny, Concurrency: -1, Confidence: 0, MinBids: 0},
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: -1, MinBids: 0},
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: 0, MinBids: -1},
	{TargetingMode: TargetAny, Concurrency: 1, Confidence: 2, MinBids: 0},
}

func TestDealValidity(t *testing.T) {
	for _, deal := range validDeals {
		t.Run(
			fmt.Sprintf("%v is valid", deal),
			func(t *testing.T) {
				require.NoError(t, deal.IsValid())
			},
		)
	}

	for _, deal := range invalidDeals {
		t.Run(
			fmt.Sprintf("%v is invalid", deal),
			func(t *testing.T) {
				require.Error(t, deal.IsValid())
			},
		)
	}
}
