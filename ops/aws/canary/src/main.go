package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/rs/zerolog/log"
	"os/exec"
)

func hello() (string, error) {
	out, err := exec.Command("ls", "-l").Output()
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("ls -l")
	log.Info().Msgf(string(out))

	out, err = exec.Command("pwd").Output()
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("pwd")
	log.Info().Msgf(string(out))

	out, err = exec.Command("./bacalhau", "version").Output()
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("bacalhau")
	log.Info().Msgf(string(out))

	jobs, err := bacalhau.GetAPIClient().List(context.Background())
	if err != nil {
		return "", err
	}

	count := 0
	for _, j := range jobs {
		log.Info().Msgf("Job: %s", j.ID)
		count++
		if count > 10 {
			break
		}
	}
	
	return "Done λ!", nil
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(hello)
}
