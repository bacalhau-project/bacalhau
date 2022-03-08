package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	logLevelString := strings.ToLower(os.Getenv("DEBUG_LEVEL"))
	fmt.Printf("Log level string: %s", logLevelString)
}
