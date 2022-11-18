package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
)

// variable to store the slack webhooks to retrieve them once and reuse them across recent invocations
var slackSecret slackSecretType

func init() {
	logger.SetupCWLogger()
	slackSecret = mustGetSlackSecret()
}

func handle(event events.SNSEvent) error {
	for _, record := range event.Records {
		cwAlarm := &events.CloudWatchAlarmSNSPayload{}
		err := json.Unmarshal([]byte(record.SNS.Message), cwAlarm)
		if err != nil {
			return fmt.Errorf("failed to unmarshal sns message %+v, to CloudWatchAlarmSNSPayload with error: %w", record, err)
		}

		slackMessage := NewSlackMessageFromEvent(cwAlarm, os.Getenv("DASHBOARD_URL"))
		marshalledMsg, err := json.Marshal(slackMessage)
		if err != nil {
			return err
		}

		resp, err := http.Post(slackSecret.WebhookUrl, "application/json", bytes.NewBuffer(marshalledMsg))
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to send message: %s, to slack due to %s\n", string(marshalledMsg), resp.Status)
		}
	}
	return nil
}

func main() {
	lambda.Start(handle)
}
