package main

import (
	"github.com/aws/aws-lambda-go/events"
)

// Type to hold slack webhooks retrieved from secret manager
type slackSecretType struct {
	WebhookUrl string `json:"webhookUrl"`
}

type slackMessage struct {
	Icon             string `json:"Icon"`
	AlarmDescription string `json:"AlarmDescription"`
	NewStateValue    string `json:"NewStateValue"`
	NewStateReason   string `json:"NewStateReason"`
	OldStateValue    string `json:"OldStateValue"`
	DashboardUrl     string `json:"DashboardUrl"`
}

func NewSlackMessageFromEvent(event *events.CloudWatchAlarmSNSPayload, dashboardUrl string) slackMessage {
	icon := ":fire:"
	if event.NewStateValue == "OK" {
		icon = ":white_check_mark:"
	}
	return slackMessage{
		Icon:             icon,
		AlarmDescription: event.AlarmDescription,
		NewStateValue:    event.NewStateValue,
		NewStateReason:   event.NewStateReason,
		OldStateValue:    event.OldStateValue,
		DashboardUrl:     dashboardUrl,
	}
}
