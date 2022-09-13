package main

import (
	"github.com/aws/aws-lambda-go/events"
)

// Type to hold slack webhooks retrieved from secret manager
type slackWebhooksType struct {
	AlarmOk        string `json:"alarmOk"`
	AlarmTriggered string `json:"alarmTriggered"`
}

type slackMessage struct {
	Icon             string `json:"icon_emoji"`
	AlarmDescription string `json:"alarmDescription"`
	NewStateValue    string `json:"newStateValue"`
	NewStateReason   string `json:"newStateReason"`
	OldStateValue    string `json:"oldStateValue"`
	DashboardUrl     string `json:"dashboardUrl"`
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
