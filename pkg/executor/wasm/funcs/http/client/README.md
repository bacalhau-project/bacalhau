# HTTP Client for Bacalhau WASM

This package provides a TinyGo-compatible client for the HTTP host functions in Bacalhau WASM executor.

## Overview

The client allows WASM modules to make HTTP requests using the host functions provided by Bacalhau. It provides a simple API for making HTTP requests and handling responses.

## Importing the Client

To use this client in your TinyGo WASM module, add it as a dependency:

```bash
# From your module directory
go get github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client
```

Then import it in your code:

```go
import "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
)

func main() {
	// Create a new HTTP client
	httpClient := client.NewClient()

	// Make a GET request
	response, err := httpClient.Get("https://example.com", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status: %d\n", response.StatusCode)
	fmt.Printf("Headers: %s\n", response.Headers)
	fmt.Printf("Body: %s\n", response.Body)
}
```

## Building Your Module with TinyGo

After importing the client, build your module with TinyGo:

```bash
tinygo build -o main.wasm -target wasi ./main.go
```

This will compile your code along with the HTTP client into a WASM module that can be executed by the Bacalhau WASM executor.

## API

### Client

```go
// Create a new HTTP client
client := client.NewClient()

// Make a GET request
response, err := client.Get(url, headers)

// Make a POST request
response, err := client.Post(url, headers, body)

// Make a PUT request
response, err := client.Put(url, headers, body)

// Make a DELETE request
response, err := client.Delete(url, headers)

// Make a custom request
response, err := client.Request(method, url, headers, body)
```

### Response

The client returns a `Response` object that contains the HTTP response:

```go
type Response struct {
	StatusCode int    // HTTP status code
	Headers    string // Response headers
	Body       string // Response body
}
```

### Constants

```go
// HTTP Methods
client.MethodGet
client.MethodPost
client.MethodPut
client.MethodDelete
client.MethodHead
client.MethodOptions
client.MethodPatch

// Status Codes
client.StatusSuccess
client.StatusInvalidURL
client.StatusNetworkError
client.StatusTimeout
client.StatusNotAllowed
client.StatusTooLarge
client.StatusBadInput
client.StatusMemoryError
```

## Limitations

- The client is designed to work with the Bacalhau WASM executor and may not work in other WASM environments.
- The client uses unsafe pointers to interact with the host functions, which may cause issues if used incorrectly.
- The client is not thread-safe and should not be used concurrently.
