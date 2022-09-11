package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/scenarios"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
)

var client *publicapi.APIClient

var testcasesMap = map[string]Handler{
	"list":         scenarios.List,
	"submit":       scenarios.Submit,
	"submitAndGet": scenarios.SubmitAndGet,
}

func init() {
	client = bacalhau.GetAPIClient()
}

func route(ctx context.Context, event Event) error {
	handler, ok := testcasesMap[event.Action]
	if !ok {
		return fmt.Errorf("no handler found for action: %s", event.Action)
	}
	err := handler(ctx, client)
	if err != nil {
		return err
	}
	return nil
}
