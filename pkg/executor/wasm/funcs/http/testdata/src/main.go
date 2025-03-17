// HTTP Test Module for Bacalhau
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
)

// getMethodConstant converts a string method to its uint32 constant
func getMethodConstant(method string) (uint32, error) {
	switch strings.ToUpper(method) {
	case "GET":
		return client.MethodGet, nil
	case "POST":
		return client.MethodPost, nil
	case "PUT":
		return client.MethodPut, nil
	case "DELETE":
		return client.MethodDelete, nil
	case "HEAD":
		return client.MethodHead, nil
	case "OPTIONS":
		return client.MethodOptions, nil
	case "PATCH":
		return client.MethodPatch, nil
	default:
		return 0, fmt.Errorf("unsupported HTTP method: %s", method)
	}
}

// parseHeaders converts a header string to Headers map
func parseHeaders(headerStr string) client.Headers {
	headers := make(client.Headers)
	if headerStr == "" {
		return headers
	}

	// Split headers by newline
	lines := strings.Split(headerStr, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers.Add(key, value)
		}
	}
	return headers
}

// main is the entry point for the WebAssembly application
func main() {
	// Get inputs from environment variables
	method := os.Getenv("HTTP_METHOD")
	url := os.Getenv("HTTP_URL")
	headerStr := os.Getenv("HTTP_HEADERS")
	body := os.Getenv("HTTP_BODY")

	// Validate URL
	if url == "" {
		fmt.Println("Error: HTTP_URL environment variable is required")
		os.Exit(1)
	}

	// Default to GET if method is not specified
	if method == "" {
		method = "GET"
	}

	// Convert method string to constant
	methodCode, err := getMethodConstant(method)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create HTTP client
	httpClient := client.NewClient()

	// Parse headers
	headers := parseHeaders(headerStr)

	// Make the request
	fmt.Printf("Making %s request to %s\n", method, url)
	response, err := httpClient.Request(methodCode, url, headers, body)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Printf("Status Code: %d\n", response.StatusCode)
	fmt.Println("Headers:")
	for key, values := range response.Headers {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
	fmt.Printf("\nBody:\n%s\n", response.Body)

	os.Exit(0)
}
