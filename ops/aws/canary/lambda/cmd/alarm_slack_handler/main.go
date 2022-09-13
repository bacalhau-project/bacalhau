package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
	"net/http"
	"os"
)

// variable to store the slack webhooks to retrieve them once and reuse them across recent invocations
var slackWebhooks slackWebhooksType

func init() {
	logger.SetupCWLogger()
	slackWebhooks = mustGetWebhookSecret()
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

		resp, err := http.Post(slackWebhooks.AlarmOk, "application/json", bytes.NewBuffer(marshalledMsg))
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("slack returned status code %d", resp.StatusCode)
		}
	}
	return nil
}

func main() {
	//cwAlarm := &events.CloudWatchAlarmSNSPayload{}
	//cwAlarm.AlarmDescription = "List Availability"
	//cwAlarm.NewStateValue = "OK"
	//cwAlarm.NewStateReason = "Threshold Crossed: 1 datapoint (0.0) was not less than or equal to the threshold (0.0)."
	//
	//marshall, err := json.Marshal(cwAlarm)
	//if err != nil {
	//	panic(err)
	//}
	//snsEvent := events.SNSEvent{
	//	Records: []events.SNSEventRecord{
	//		events.SNSEventRecord{
	//			SNS: events.SNSEntity{
	//				Message: string(marshall),
	//			},
	//		},
	//	},
	//}
	//handle(snsEvent)

	lambda.Start(handle)
}
