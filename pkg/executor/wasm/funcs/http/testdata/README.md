# HTTP Test Module

This is a test module for the HTTP client in Bacalhau. It allows making HTTP requests using environment variables to specify the method, URL, headers, and body.

## Location

This test module is located in the Bacalhau codebase under:

```
pkg/executor/wasm/funcs/http/test/
```

It tests the HTTP client implementation located at:

```
pkg/executor/wasm/funcs/http/client/
```

## Building

To build the module, run:

```bash
make
```

or to force a rebuild:

```bash
make force
```

This will compile the TinyGo source code into a WASM module.

## Dependencies

This module uses the HTTP client package from the Bacalhau codebase:

```
github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client
```

The dependency is managed through the Go workspace (go.work) at the root of the repository, which allows the test module to directly use the client package without copying it.

## Usage

The module accepts the following environment variables:

- `HTTP_METHOD`: The HTTP method to use (GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH). Defaults to GET if not specified.
- `HTTP_URL`: The URL to request (required)
- `HTTP_HEADERS`: HTTP headers to include in the request
- `HTTP_BODY`: Request body (for POST, PUT, PATCH)


## Implementation Details

The module imports and uses the HTTP client from the `github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client` package to make HTTP requests. This client package is maintained separately in the Bacalhau codebase and is not part of this test module. The test module simply demonstrates how to use the client package to make HTTP requests. It reads the environment variables, makes the appropriate request, and prints the response status code, headers, and body.

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
