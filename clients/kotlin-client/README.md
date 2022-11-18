# bacalhau-client - Kotlin client library for Bacalhau API

## Requires

* Kotlin 1.4.30
* Gradle 5.3

## Build

First, create the gradle wrapper script:

```
gradle wrapper
```

Then, run:

```
./gradlew check assemble
```

This runs all tests and packages the library.

## Features/Implementation Notes

* Supports JSON inputs/outputs, File inputs, and Form inputs.
* Supports collection formats for query parameters: csv, tsv, ssv, pipes.
* Some Kotlin and Java types are fully qualified to avoid conflicts with types defined in Swagger definitions.
* Implementation of ApiClient is intended to reduce method counts, specifically to benefit Android targets.

<a name="documentation-for-api-endpoints"></a>
## Documentation for API Endpoints

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*HealthApi* | [**apiServer/debug**](docs/HealthApi.md#apiserver/debug) | **GET** /debug | Returns debug information on what the current node is doing.
*HealthApi* | [**apiServer/healthz**](docs/HealthApi.md#apiserver/healthz) | **GET** /healthz | 
*HealthApi* | [**apiServer/livez**](docs/HealthApi.md#apiserver/livez) | **GET** /livez | 
*HealthApi* | [**apiServer/logz**](docs/HealthApi.md#apiserver/logz) | **GET** /logz | 
*HealthApi* | [**apiServer/readyz**](docs/HealthApi.md#apiserver/readyz) | **GET** /readyz | 
*HealthApi* | [**apiServer/varz**](docs/HealthApi.md#apiserver/varz) | **GET** /varz | 
*JobApi* | [**pkg/apiServer.submit**](docs/JobApi.md#pkg/apiserver.submit) | **POST** /submit | Submits a new job to the network.
*JobApi* | [**pkg/publicapi.list**](docs/JobApi.md#pkg/publicapi.list) | **POST** /list | Simply lists jobs.
*JobApi* | [**pkg/publicapi/events**](docs/JobApi.md#pkg/publicapi/events) | **POST** /events | Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
*JobApi* | [**pkg/publicapi/localEvents**](docs/JobApi.md#pkg/publicapi/localevents) | **POST** /local_events | Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
*JobApi* | [**pkg/publicapi/results**](docs/JobApi.md#pkg/publicapi/results) | **POST** /results | Returns the results of the job-id specified in the body payload.
*JobApi* | [**pkg/publicapi/states**](docs/JobApi.md#pkg/publicapi/states) | **POST** /states | Returns the state of the job-id specified in the body payload.
*MiscApi* | [**apiServer/id**](docs/MiscApi.md#apiserver/id) | **GET** /id | Returns the id of the host node.
*MiscApi* | [**apiServer/peers**](docs/MiscApi.md#apiserver/peers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
*MiscApi* | [**apiServer/version**](docs/MiscApi.md#apiserver/version) | **POST** /version | Returns the build version running on the server.

<a name="documentation-for-models"></a>
## Documentation for Models

 - [io.swagger.client.models.ComputenodeActiveJob](docs/ComputenodeActiveJob.md)
 - [io.swagger.client.models.ModelBuildVersionInfo](docs/ModelBuildVersionInfo.md)
 - [io.swagger.client.models.ModelDeal](docs/ModelDeal.md)
 - [io.swagger.client.models.ModelJob](docs/ModelJob.md)
 - [io.swagger.client.models.ModelJobCreatePayload](docs/ModelJobCreatePayload.md)
 - [io.swagger.client.models.ModelJobEvent](docs/ModelJobEvent.md)
 - [io.swagger.client.models.ModelJobExecutionPlan](docs/ModelJobExecutionPlan.md)
 - [io.swagger.client.models.ModelJobLocalEvent](docs/ModelJobLocalEvent.md)
 - [io.swagger.client.models.ModelJobNodeState](docs/ModelJobNodeState.md)
 - [io.swagger.client.models.ModelJobShardState](docs/ModelJobShardState.md)
 - [io.swagger.client.models.ModelJobShardingConfig](docs/ModelJobShardingConfig.md)
 - [io.swagger.client.models.ModelJobSpecDocker](docs/ModelJobSpecDocker.md)
 - [io.swagger.client.models.ModelJobSpecLanguage](docs/ModelJobSpecLanguage.md)
 - [io.swagger.client.models.ModelJobSpecWasm](docs/ModelJobSpecWasm.md)
 - [io.swagger.client.models.ModelJobState](docs/ModelJobState.md)
 - [io.swagger.client.models.ModelPublishedResult](docs/ModelPublishedResult.md)
 - [io.swagger.client.models.ModelResourceUsageConfig](docs/ModelResourceUsageConfig.md)
 - [io.swagger.client.models.ModelResourceUsageData](docs/ModelResourceUsageData.md)
 - [io.swagger.client.models.ModelRunCommandResult](docs/ModelRunCommandResult.md)
 - [io.swagger.client.models.ModelSpec](docs/ModelSpec.md)
 - [io.swagger.client.models.ModelStorageSpec](docs/ModelStorageSpec.md)
 - [io.swagger.client.models.ModelVerificationResult](docs/ModelVerificationResult.md)
 - [io.swagger.client.models.PublicapidebugResponse](docs/PublicapidebugResponse.md)
 - [io.swagger.client.models.PublicapieventsRequest](docs/PublicapieventsRequest.md)
 - [io.swagger.client.models.PublicapieventsResponse](docs/PublicapieventsResponse.md)
 - [io.swagger.client.models.PublicapilistRequest](docs/PublicapilistRequest.md)
 - [io.swagger.client.models.PublicapilistResponse](docs/PublicapilistResponse.md)
 - [io.swagger.client.models.PublicapilocalEventsRequest](docs/PublicapilocalEventsRequest.md)
 - [io.swagger.client.models.PublicapilocalEventsResponse](docs/PublicapilocalEventsResponse.md)
 - [io.swagger.client.models.PublicapiresultsResponse](docs/PublicapiresultsResponse.md)
 - [io.swagger.client.models.PublicapistateRequest](docs/PublicapistateRequest.md)
 - [io.swagger.client.models.PublicapistateResponse](docs/PublicapistateResponse.md)
 - [io.swagger.client.models.PublicapisubmitRequest](docs/PublicapisubmitRequest.md)
 - [io.swagger.client.models.PublicapisubmitResponse](docs/PublicapisubmitResponse.md)
 - [io.swagger.client.models.PublicapiversionRequest](docs/PublicapiversionRequest.md)
 - [io.swagger.client.models.PublicapiversionResponse](docs/PublicapiversionResponse.md)
 - [io.swagger.client.models.RequesternodeActiveJob](docs/RequesternodeActiveJob.md)
 - [io.swagger.client.models.TypesFreeSpace](docs/TypesFreeSpace.md)
 - [io.swagger.client.models.TypesHealthInfo](docs/TypesHealthInfo.md)
 - [io.swagger.client.models.TypesMountStatus](docs/TypesMountStatus.md)

<a name="documentation-for-authorization"></a>
## Documentation for Authorization

All endpoints do not require authorization.
