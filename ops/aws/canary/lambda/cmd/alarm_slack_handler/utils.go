package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"os"
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
