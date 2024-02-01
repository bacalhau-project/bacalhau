package main

import (
	"github.com/bacalhau-project/bacalhau/apps/job-info-consumer/consumer/cmd"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
