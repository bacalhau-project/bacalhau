package main

import (
	"github.com/filecoin-project/bacalhau/dashboard/api/cmd/dashboard"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	dashboard.Execute()
}
