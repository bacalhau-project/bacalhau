package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func hello() (string, error) {
	err := system.InitConfig()
	if err != nil {
		return "", err
	}
	return "Done λ!", nil
}

func main() {
	lambda.Start(hello)
}
