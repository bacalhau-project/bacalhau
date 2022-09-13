package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"os"
)

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

func createSlackMessageFromEvent(event events.CloudWatchAlarmSNSPayload) slackMessage {
	dashboardUrl := os.Getenv("DASHBOARD_URL")
	icon := ":fire:"
	if event.NewStateValue == "OK" {
		icon = ":white_check_mark:"
	}

	text := `%s *%s* is now *%s*: %s
Check the [dashboard](%s) for more information.
`
	return slackMessage{
		Text: fmt.Sprintf(text, icon, event.AlarmName, event.NewStateValue, event.NewStateReason, dashboardUrl),
	}
}
