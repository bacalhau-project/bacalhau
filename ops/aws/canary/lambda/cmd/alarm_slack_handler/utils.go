package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/slack-go/slack"
	"os"
	"strconv"
	"time"
)

func mustGetSlackSecret() slackSecretType {
	secretName := os.Getenv("SLACK_SECRET_NAME")

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

	newSlackWebhooks := slackSecretType{}
	err = json.Unmarshal([]byte(*result.SecretString), &newSlackWebhooks)
	if err != nil {
		panic(err)
	}

	return newSlackWebhooks
}

func NewSlackMessageFromEvent(event *events.CloudWatchAlarmSNSPayload, dashboardUrl string) slack.Msg {
	color := "danger"
	if event.NewStateValue == "OK" {
		color = "good"
	}
	attachment := slack.Attachment{
		Color: color,
		Fields: []slack.AttachmentField{
			{"Alarm Description", event.AlarmDescription, false},
			{"New Liveness Reason", event.NewStateReason, false},
			{"Old Liveness", event.OldStateValue, true},
			{"New Liveness", event.NewStateValue, true},
		},
		Actions: []slack.AttachmentAction{
			{Name: "View Dashboard", Type: "button", Text: "View Dashboard", URL: dashboardUrl},
		},
		Ts: json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	return slack.Msg{
		Text:        "*Bacalhau Canary Notification*",
		Attachments: []slack.Attachment{attachment},
	}
}
