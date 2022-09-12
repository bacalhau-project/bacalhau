package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
	"os"
	"strings"
)

func init() {
	sess := session.Must(session.NewSession())
	secretsmanager.New(sess)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		fmt.Println(pair[0])
	}
}

func handle(event events.CloudWatchAlarmSNSPayload) error {
	fmt.Println(event)
	return nil
}

func init() {
	logger.SetupCWLogger()
}

func main() {
	lambda.Start(handle)
}
