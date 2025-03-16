# HTTP Client for Bacalhau WASM

This package provides a TinyGo-compatible client for the HTTP host functions in Bacalhau WASM executor.

## Table of Contents

- [Overview](#overview)
- [Importing the Client](#importing-the-client)
- [Basic Usage](#basic-usage)
- [Building Your Module](#building-your-module-with-tinygo)
- [Running Your Module](#running-your-module)
- [API Reference](#api-reference)
  - [Client Methods](#client-methods)
  - [Response Object](#response-object)
  - [Constants](#constants)
- [Advanced Usage](#advanced-usage)
  - [Error Handling](#error-handling)
  - [Custom Headers](#custom-headers)
  - [Configuring Buffer Sizes](#configuring-buffer-sizes)
- [Development](#development)
- [Example Projects](#example-projects)
- [Security Considerations](#security-considerations)
- [Limitations](#limitations)

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

## Basic Usage

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

## Running Your Module

Once you've built your module, you can run it with the Bacalhau CLI:

```bash
bacalhau wasm run main.wasm
```

## API Reference

### Client Methods

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

### Response Object

The client returns a `Response` object that contains the HTTP response:

```go
type Response struct {
	StatusCode int    // HTTP status code
	Headers    string // Response headers
	Body       string // Response body
}
```

You can access these fields directly:

```go
response, err := httpClient.Get(url, headers)
if err != nil {
	// Handle error
}

fmt.Printf("Status: %d\n", response.StatusCode)
fmt.Printf("Headers: %s\n", response.Headers)
fmt.Printf("Body: %s\n", response.Body)
```

### Constants

```go
// HTTP Methods
client.MethodGet     // 0
client.MethodPost    // 1
client.MethodPut     // 2
client.MethodDelete  // 3
client.MethodHead    // 4
client.MethodOptions // 5
client.MethodPatch   // 6

// Status Codes
client.StatusSuccess      // 0 - Request succeeded
client.StatusInvalidURL   // 1 - Invalid URL format
client.StatusNetworkError // 2 - Network connection error
client.StatusTimeout      // 3 - Request timed out
client.StatusNotAllowed   // 4 - Host not allowed (security restriction)
client.StatusTooLarge     // 5 - Response too large for buffer
client.StatusBadInput     // 6 - Invalid input parameters
client.StatusMemoryError  // 7 - Memory allocation/access error
```

## Advanced Usage

### Error Handling

The client returns an error if the request fails:

```go
response, err := httpClient.Get(url, headers)
if err != nil {
	// Handle error
	fmt.Printf("Error: %v\n", err)
	return
}
```

The error will be of type `*HTTPError` if it's a specific HTTP error:

```go
if httpErr, ok := err.(*client.HTTPError); ok {
	fmt.Printf("HTTP Error: %s (Code: %d)\n", httpErr.Message, httpErr.Code)
}
```

### Custom Headers

You can include custom headers in your requests:

```go
// Single header
headers := "Content-Type: application/json"

// Multiple headers (separated by newlines)
headers := "Content-Type: application/json\nAuthorization: Bearer token123"

response, err := httpClient.Post(url, headers, body)
```

### Configuring Buffer Sizes

You can configure the buffer sizes for response headers and body:

```go
// Create a client with custom buffer sizes
httpClient := &client.Client{
	HeadersBufferSize: 16 * 1024,  // 16KB for headers
	BodyBufferSize:    1024 * 1024, // 1MB for body
}

// Default buffer sizes
// HeadersBufferSize: 8 * 1024   (8KB)
// BodyBufferSize:    256 * 1024 (256KB)
```

## Development

### Building the Client Module

The HTTP client module is designed to be imported as a Go package, not compiled as a standalone WASM module. However, you can verify that it compiles correctly with TinyGo:

```bash
# From the client directory
make check
```

This will compile the client module with TinyGo and output a WASM file in the `build` directory to verify compilation. This WASM file is not meant to be used directly, but rather to verify that the client module compiles correctly with TinyGo.

## Example Projects

For a complete example of using this client, see the HTTP test module in the Bacalhau repository:

```
pkg/executor/wasm/funcs/http/test/
```

This module demonstrates how to use the HTTP client to make various types of requests and handle responses.

### Example: JSON API Request

```go
package main

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
)

func main() {
	httpClient := client.NewClient()

	url := "https://api.example.com/data"
	headers := "Content-Type: application/json\nAccept: application/json"
	body := `{"query": "example"}`

	response, err := httpClient.Post(url, headers, body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status: %d\n", response.StatusCode)
	fmt.Printf("Response: %s\n", response.Body)
}
```

## Security Considerations

When using the HTTP client, be aware of these security considerations:

1. **Host Restrictions**: The Bacalhau executor may restrict which hosts your WASM module can connect to. Requests to unauthorized hosts will fail with `StatusNotAllowed`.

2. **Data Handling**: Be careful when processing response data, especially when parsing JSON or other formats that could contain malicious content.

3. **Credentials**: Avoid hardcoding sensitive information like API keys or passwords in your WASM modules. Use environment variables or other secure methods to provide credentials.

4. **TLS Verification**: The HTTP client performs TLS verification by default. This helps protect against man-in-the-middle attacks.

## Limitations

- The client is designed to work with the Bacalhau WASM executor and may not work in other WASM environments.
- The client uses unsafe pointers to interact with the host functions, which may cause issues if used incorrectly.
- The client is not thread-safe and should not be used concurrently.
- Maximum response sizes are limited by the buffer sizes configured in the client.
- Only string-based headers are supported; structured header objects are not available.
