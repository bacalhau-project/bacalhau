package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/logger"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/router"
)

func init() {
	logger.SetupCWLogger()
}

func main() {
	// running in lambda
	lambda.Start(router.Route)
}
