package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func hello() (string, error) {
	jobs, err := bacalhau.GetAPIClient().List(context.Background())
	if err != nil {
		return "", err
	}

	count := 0
	for _, j := range jobs {
		fmt.Printf("Job: %s\n", j.ID)
		count++
		if count > 10 {
			break
		}
	}

	return "Done λ!", nil
}

func init() {
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	//_, err := hello()
	//if err != nil {
	//	return
	//}
	lambda.Start(hello)
}
