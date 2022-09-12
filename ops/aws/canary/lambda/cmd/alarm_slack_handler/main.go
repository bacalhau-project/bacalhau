package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
)

func handle(event events.CloudWatchAlarmSNSPayload) error {
	fmt.Println(event)
	return nil
}

func init() {
	logger.SetupCWLogger()
}

func main() {
	lambda.Start(handle)
}
