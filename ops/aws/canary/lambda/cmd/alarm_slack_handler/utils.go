package main

import (
	"encoding/json"
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
