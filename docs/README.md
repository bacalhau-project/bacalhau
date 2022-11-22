# docs

## Codebase docs

* [Running locally](./running_locally.md)
* [Debugging locally](./debugging_locally.md)
* [Traceability: Open Telemetry in Bacalhau](./open_telemetry_in_bacalhau.md)

## Swagger docs

* `swagger/`: these markdown files contain long-form descriptions of the API endpoints.
* `docs.go`, `swagger.json`, `swagger.yaml`: swagger specification of the API.

Build the swagger specification files with `make swagger-docs`.