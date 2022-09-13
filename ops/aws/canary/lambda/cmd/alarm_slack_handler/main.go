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
	//slackWebhooks = mustGetWebhookSecret()
}

func handle(event events.SNSEvent) error {
	fmt.Printf("Received event: %+v\n", event)
	for _, record := range event.Records {
		fmt.Printf("SNS record: %+v\n", record)
	}
	return nil
}

func main() {
	//	url := "https://hooks.slack.com/workflows/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"
	//
	//	md := `:fire: alarm state is now *ALARM*
	//*Alarm Name*: test
	//*Alarm Description*: test
	//*AWS Account ID*: 425283959824`
	//
	//	message := slackMessage{
	//		Text: md,
	//	}
	//	marshal, err := json.Marshal(message)
	//	if err != nil {
	//		panic(err)
	//	}
	//	resp, err := http.Post(url, "application/json", bytes.NewBuffer(marshal))
	//
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer resp.Body.Close()
	//
	//	fmt.Println("response Status:", resp.Status)
	//	fmt.Println("response Headers:", resp.Header)
	//	body, _ := ioutil.ReadAll(resp.Body)
	//	fmt.Println("response Body:", string(body))

	lambda.Start(handle)
}
