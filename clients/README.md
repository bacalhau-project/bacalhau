# Autogenerate Bacalhau clients

This folder contains the Makefile and config files to use [Swagger](https://swagger.io/tools/swagger-codegen/)/OpenAPI to auto-generate Bacalhau clients for the programming languages listed in [clients/supported_langs](clients/supported_langs).
These clients wrap *only* the API endpoint calls and request/response models and do not ship the client-side logic necessary to properly operate the endpoints. It's highly recommended to use the [Bacalhau Python SDK instead](../python).

---

~Note: for some reason, `swagger-codegen` version 3.0.36 does not generate any model nor API files properly.
Please use version 2.4.29 instead.~
