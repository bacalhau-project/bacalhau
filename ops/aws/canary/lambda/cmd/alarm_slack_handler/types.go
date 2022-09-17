package main

// Type to hold slack webhooks retrieved from secret manager
type slackSecretType struct {
	WebhookUrl string `json:"webhookUrl"`
}
