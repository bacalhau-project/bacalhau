# docs

## Codebase docs

* [Running locally](./running_locally.md)
* [Debugging locally](./debugging_locally.md)
* [Traceability: Open Telemetry in Bacalhau](./open_telemetry_in_bacalhau.md)

## OpenAPI docs

OpenAPI annotations sit next to the endpoints in `pkg/publicapi`; these are built using [swag](https://github.com/swaggo/swag), a Go converter for Swagger documentation.
Find more details about the Swag annotations [in their docs](https://github.com/swaggo/swag#declarative-comments-format).

* `docs.go`, `swagger.json`, `swagger.yaml`: swagger specification of the API.
* `swagger/`: these markdown files contain long-form descriptions of the API endpoints.

The swagger specification is built automatically by the CI pipeline (see the `build_swagger` workflow).
You can build them locally with `make swagger-docs`.