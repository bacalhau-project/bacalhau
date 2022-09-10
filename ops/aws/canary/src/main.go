package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

func hello() (string, error) {
	return "Done λ!", nil
}

func main() {
	lambda.Start(hello)
}
