# Guide: Building and Using the HTTP Client Module

This guide explains how to build the HTTP client module and use it in your own TinyGo WASM modules.

## Building the Client Module

The HTTP client module is designed to be imported as a Go package, not compiled as a standalone WASM module. However, we still need to ensure it compiles correctly with TinyGo.

To check if the client module compiles correctly:

```bash
# From the client directory
make check
```

This will compile the client module with TinyGo and output a WASM file in the `build` directory. This WASM file is not meant to be used directly, but rather to verify that the client module compiles correctly with TinyGo.

## Using the Client Module in Your Own Module

To use the HTTP client module in your own TinyGo WASM module, follow these steps:

1. Import the client package in your Go code:

```go
import "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client"
```

2. Create a client instance and use it to make HTTP requests:

```go
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
```

3. Build your module with TinyGo:

```bash
tinygo build -o main.wasm -target wasi .
```

## Running Your Module

Once you've built your module, you can run it with the Bacalhau CLI:

```bash
bacalhau wasm run main.wasm
```

## Advanced Usage

### Making Different Types of Requests

The HTTP client supports various HTTP methods:

```go
// GET request
response, err := httpClient.Get(url, headers)

// POST request
response, err := httpClient.Post(url, headers, body)

// PUT request
response, err := httpClient.Put(url, headers, body)

// DELETE request
response, err := httpClient.Delete(url, headers)

// Custom request
response, err := httpClient.Request(method, url, headers, body)
```

### Working with Response Objects

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
