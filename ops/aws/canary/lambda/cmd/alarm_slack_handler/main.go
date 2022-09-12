package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
	"os"
)

// Type to hold slack webhooks retrieved from secret manager
type slackWebhooksType struct {
	AlarmOk        string `json:"alarmOk"`
	AlarmTriggered string `json:"alarmTriggered"`
}

// variable to store the slack webhooks to retrieve them once and reuse them across recent invocations
var slackWebhooks = mustGetWebhookSecret()

func mustGetWebhookSecret() slackWebhooksType {
	secretName := os.Getenv("SLACK_WEBHOOK_SECRET_NAME")

	//Create a Secrets Manager client
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		panic(err)
	}

	newSlackWebhooks := slackWebhooksType{}
	err = json.Unmarshal([]byte(*result.SecretString), &newSlackWebhooks)
	if err != nil {
		panic(err)
	}

	return newSlackWebhooks
}

func init() {
	logger.SetupCWLogger()
}

func handle(event events.CloudWatchAlarmSNSPayload) error {
	fmt.Println(event)
	fmt.Printf("Webhooks: %+v\n", slackWebhooks)
	return nil
}

func main() {
	lambda.Start(handle)
}
