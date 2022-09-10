package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
)

func hello() (string, error) {
	bacalhau.GetAPIClient()
	return "Done λ!", nil
}

func main() {
	lambda.Start(hello)
}
