package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handle(event events.CloudWatchAlarmSNSPayload) error {
	fmt.Println(event)
	return nil
}

func main() {
	lambda.Start(handle)
}
