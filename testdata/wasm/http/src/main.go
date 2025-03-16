// HTTP Test Module for Bacalhau
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
)

func main() {
	// Get method and URL from environment variables
	method := os.Getenv("HTTP_METHOD")
	url := os.Getenv("HTTP_URL")
	headers := os.Getenv("HTTP_HEADERS")
	body := os.Getenv("HTTP_BODY")

	// Validate inputs
	if url == "" {
		fmt.Println("Error: HTTP_URL environment variable is required")
		os.Exit(1)
	}

	// Default to GET if method is not specified
	if method == "" {
		method = "GET"
	}

	// Create HTTP client
	httpClient := client.NewClient()

	// Make the request based on the method
	var response *client.Response
	var err error

	fmt.Printf("Making %s request to %s\n", method, url)

	switch strings.ToUpper(method) {
	case "GET":
		response, err = httpClient.Get(url, headers)
	case "POST":
		response, err = httpClient.Post(url, headers, body)
	case "PUT":
		response, err = httpClient.Put(url, headers, body)
	case "DELETE":
		response, err = httpClient.Delete(url, headers)
	default:
		// Convert string method to uint32 constant
		var methodCode uint32
		switch strings.ToUpper(method) {
		case "HEAD":
			methodCode = client.MethodHead
		case "OPTIONS":
			methodCode = client.MethodOptions
		case "PATCH":
			methodCode = client.MethodPatch
		default:
			fmt.Printf("Error: Unsupported HTTP method: %s\n", method)
			os.Exit(1)
		}
		response, err = httpClient.Request(methodCode, url, headers, body)
	}

	// Handle errors
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Printf("Status Code: %d\n", response.StatusCode)
	fmt.Printf("Headers:\n%s\n", response.Headers)
	fmt.Printf("Body:\n%s\n", response.Body)

	os.Exit(0)
}
