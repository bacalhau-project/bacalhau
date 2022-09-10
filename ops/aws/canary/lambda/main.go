package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func hello(event events.CloudWatchEvent) (string, error) {
	fmt.Printf("Received event: %v", event)

	s, serdeErr := json.Marshal(event)
	if serdeErr != nil {
		return "", serdeErr
	}
	fmt.Printf("Event Jsong: %v", s)

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
	lambda.Start(hello)
}
