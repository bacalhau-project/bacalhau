# HTTP Test Module

This is a test module for the HTTP client in Bacalhau. It allows making HTTP requests using environment variables to specify the method, URL, headers, and body.

## Building

To build the module, run:

```bash
make
```

This will:

1. Tidy the Go module dependencies
2. Compile the TinyGo source code into a WASM module
3. Generate a Go file that embeds the WASM module

## Dependencies

This module depends only on the HTTP client package:

```
github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client
```

## Usage

The module accepts the following environment variables:

- `HTTP_METHOD`: The HTTP method to use (GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH). Defaults to GET if not specified.
- `HTTP_URL`: The URL to request (required)
- `HTTP_HEADERS`: HTTP headers to include in the request
- `HTTP_BODY`: Request body (for POST, PUT, PATCH)

### Example

```bash
# Make a GET request
HTTP_URL=https://example.com bacalhau wasm run main.wasm

# Make a POST request with headers and body
HTTP_METHOD=POST \
HTTP_URL=https://example.com/api \
HTTP_HEADERS="Content-Type: application/json" \
HTTP_BODY='{"key": "value"}' \
bacalhau wasm run main.wasm
```

## Implementation Details

The module uses the HTTP client from the `github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client` package to make HTTP requests. It reads the environment variables, makes the appropriate request, and prints the response status code, headers, and body.

## Output

The module outputs:

- The request method and URL
- The response status code
- The response headers
- The response body

Example output:

```
Making GET request to https://example.com
Status Code: 200
Headers:
Content-Type: text/html; charset=UTF-8
Content-Length: 1234
...
Body:
<!doctype html>
<html>
...
</html>
```
