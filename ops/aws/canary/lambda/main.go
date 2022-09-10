package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/scenarios"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

var client *publicapi.APIClient
var testcasesMap = map[string]Handler{
	"list": scenarios.List,
}

func init() {
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
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

func main() {
	lambda.Start(route)
}
