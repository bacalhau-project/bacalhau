package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
)

// variable to store the slack webhooks to retrieve them once and reuse them across recent invocations
var slackWebhooks slackWebhooksType

func init() {
	logger.SetupCWLogger()
	slackWebhooks = mustGetWebhookSecret()
}

func handle(event events.SNSEvent) error {
	for _, record := range event.Records {
		fmt.Printf("SNS record: %+v\n", record)
	}
	return nil
}

func main() {
	lambda.Start(handle)
}
