package main

// Type to hold slack webhooks retrieved from secret manager
type slackWebhooksType struct {
	AlarmOk        string `json:"alarmOk"`
	AlarmTriggered string `json:"alarmTriggered"`
}

type slackMessage struct {
	Text string `json:"text"`
}
