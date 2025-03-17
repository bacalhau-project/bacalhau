# WASM HTTP Client

A lightweight, TinyGo-compatible HTTP client for WebAssembly modules. This client provides a simple and efficient way to make HTTP requests from WASM modules using host functions.

## Features

- Simple, straightforward API
- Support for all standard HTTP methods (GET, POST, PUT, DELETE, etc.)
- Type-safe header management
- Configurable buffer sizes for headers and body
- TinyGo compatibility
- Error handling with specific error types

## Usage

### Basic Examples

```go
// Create a new client
client := NewClient()

// Simple GET request
resp, err := client.Get("https://api.example.com", nil)
if err != nil {
    // Handle error
}
fmt.Printf("Status: %d\nBody: %s\n", resp.StatusCode, resp.Body)

// POST request with headers and body
headers := make(Headers)
headers.Set("Content-Type", "application/json")
headers.Set("Authorization", "Bearer token123")

resp, err = client.Post(
    "https://api.example.com/data",
    headers,
    `{"key": "value"}`,
)
```

### Working with Headers

```go
headers := make(Headers)

// Add multiple values for the same header
headers.Add("Accept", "application/json")
headers.Add("Accept", "text/plain")

// Set (replace) header value
headers.Set("Content-Type", "application/json")

// Get first header value
contentType := headers.Get("Content-Type")

// Get all values for a header
accepts := headers.Values("Accept")
```

### All Available Methods

```go
client := NewClient()

// GET request
resp, err := client.Get(url, headers)

// POST request
resp, err := client.Post(url, headers, body)

// PUT request
resp, err := client.Put(url, headers, body)

// DELETE request
resp, err := client.Delete(url, headers)

// HEAD request
resp, err := client.Head(url, headers)

// OPTIONS request
resp, err := client.Options(url, headers)

// PATCH request
resp, err := client.Patch(url, headers, body)

// Generic request (if needed)
resp, err := client.Request(method, url, headers, body)
```

### Response Handling

```go
resp, err := client.Get("https://api.example.com", nil)
if err != nil {
    switch e := err.(type) {
    case *HTTPError:
        fmt.Printf("HTTP error: %s (code: %d)\n", e.Message, e.Code)
    default:
        fmt.Printf("Error: %v\n", err)
    }
    return
}

// Access response data
fmt.Printf("Status Code: %d\n", resp.StatusCode)
fmt.Printf("Body: %s\n", resp.Body)

// Access response headers
contentType := resp.Headers.Get("Content-Type")
```

## Configuration

The client supports configurable buffer sizes for headers and body:

```go
client := &Client{
    HeadersBufferSize: 16 * 1024,  // 16KB for headers
    BodyBufferSize:    512 * 1024, // 512KB for body
}
```

Default buffer sizes:

- Headers: 8KB
- Body: 256KB

## Error Handling

The client returns specific error types for different failure scenarios:

- `StatusInvalidURL`: Invalid URL format or structure
- `StatusNetworkError`: Network-related errors
- `StatusTimeout`: Request timed out
- `StatusNotAllowed`: Host not allowed by security policy
- `StatusTooLarge`: Response too large for buffer
- `StatusBadInput`: Invalid input parameters
- `StatusMemoryError`: Memory allocation or access error

## Environment Variables (Test Program)

When using the test program, configure the HTTP request using these environment variables:

- `HTTP_METHOD`: The HTTP method to use (GET, POST, etc.)
- `HTTP_URL`: The target URL (required)
- `HTTP_HEADERS`: Headers in "Key: Value" format, one per line
- `HTTP_BODY`: Request body content

Example:

```bash
export HTTP_URL="https://api.example.com"
export HTTP_METHOD="POST"
export HTTP_HEADERS="Content-Type: application/json\nAuthorization: Bearer token123"
export HTTP_BODY='{"key": "value"}'
```
