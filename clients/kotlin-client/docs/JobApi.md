# JobApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**pkg/apiServer.submit**](JobApi.md#pkg/apiServer.submit) | **POST** /submit | Submits a new job to the network.
[**pkg/publicapi.list**](JobApi.md#pkg/publicapi.list) | **POST** /list | Simply lists jobs.
[**pkg/publicapi/events**](JobApi.md#pkg/publicapi/events) | **POST** /events | Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
[**pkg/publicapi/localEvents**](JobApi.md#pkg/publicapi/localEvents) | **POST** /local_events | Returns the node&#x27;s local events related to the job-id passed in the body payload. Useful for troubleshooting.
[**pkg/publicapi/results**](JobApi.md#pkg/publicapi/results) | **POST** /results | Returns the results of the job-id specified in the body payload.
[**pkg/publicapi/states**](JobApi.md#pkg/publicapi/states) | **POST** /states | Returns the state of the job-id specified in the body payload.

<a name="pkg/apiServer.submit"></a>
# **pkg/apiServer.submit**
> PublicapisubmitResponse pkg/apiServer.submit(body)

Submits a new job to the network.

Description:  * &#x60;client_public_key&#x60;: The base64-encoded public key of the client. * &#x60;signature&#x60;: A base64-encoded signature of the &#x60;data&#x60; attribute, signed by the client. * &#x60;data&#x60;     * &#x60;ClientID&#x60;: Request must specify a &#x60;ClientID&#x60;. To retrieve your &#x60;ClientID&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field.     * &#x60;Job&#x60;: see example below.  Example request &#x60;&#x60;&#x60;json {  \&quot;data\&quot;: {   \&quot;ClientID\&quot;: \&quot;ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\&quot;,   \&quot;Job\&quot;: {    \&quot;APIVersion\&quot;: \&quot;V1beta1\&quot;,    \&quot;Spec\&quot;: {     \&quot;Engine\&quot;: \&quot;Docker\&quot;,     \&quot;Verifier\&quot;: \&quot;Noop\&quot;,     \&quot;Publisher\&quot;: \&quot;Estuary\&quot;,     \&quot;Docker\&quot;: {      \&quot;Image\&quot;: \&quot;ubuntu\&quot;,      \&quot;Entrypoint\&quot;: [       \&quot;date\&quot;      ]     },     \&quot;Timeout\&quot;: 1800,     \&quot;outputs\&quot;: [      {       \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,       \&quot;Name\&quot;: \&quot;outputs\&quot;,       \&quot;path\&quot;: \&quot;/outputs\&quot;      }     ],     \&quot;Sharding\&quot;: {      \&quot;BatchSize\&quot;: 1,      \&quot;GlobPatternBasePath\&quot;: \&quot;/inputs\&quot;     }    },    \&quot;Deal\&quot;: {     \&quot;Concurrency\&quot;: 1    }   }  },  \&quot;signature\&quot;: \&quot;...\&quot;,  \&quot;client_public_key\&quot;: \&quot;...\&quot; } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapisubmitRequest =  // PublicapisubmitRequest | 
try {
    val result : PublicapisubmitResponse = apiInstance.pkg/apiServer.submit(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/apiServer.submit")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/apiServer.submit")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapisubmitRequest**](PublicapisubmitRequest.md)|  |

### Return type

[**PublicapisubmitResponse**](PublicapisubmitResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

<a name="pkg/publicapi.list"></a>
# **pkg/publicapi.list**
> PublicapilistResponse pkg/publicapi.list(body)

Simply lists jobs.

Returns the first (sorted) #&#x60;max_jobs&#x60; jobs that belong to the &#x60;client_id&#x60; passed in the body payload (by default). If &#x60;return_all&#x60; is set to true, it returns all jobs on the Bacalhau network.  If &#x60;id&#x60; is set, it returns only the job with that ID.  Example response: &#x60;&#x60;&#x60;json {   \&quot;jobs\&quot;: [     {       \&quot;APIVersion\&quot;: \&quot;V1beta1\&quot;,       \&quot;ID\&quot;: \&quot;9304c616-291f-41ad-b862-54e133c0149e\&quot;,       \&quot;RequesterNodeID\&quot;: \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,       \&quot;RequesterPublicKey\&quot;: \&quot;...\&quot;,       \&quot;ClientID\&quot;: \&quot;ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\&quot;,       \&quot;Spec\&quot;: {         \&quot;Engine\&quot;: \&quot;Docker\&quot;,         \&quot;Verifier\&quot;: \&quot;Noop\&quot;,         \&quot;Publisher\&quot;: \&quot;Estuary\&quot;,         \&quot;Docker\&quot;: {           \&quot;Image\&quot;: \&quot;ubuntu\&quot;,           \&quot;Entrypoint\&quot;: [             \&quot;date\&quot;           ]         },         \&quot;Language\&quot;: {           \&quot;JobContext\&quot;: {}         },         \&quot;Wasm\&quot;: {},         \&quot;Resources\&quot;: {           \&quot;GPU\&quot;: \&quot;\&quot;         },         \&quot;Timeout\&quot;: 1800,         \&quot;outputs\&quot;: [           {             \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,             \&quot;Name\&quot;: \&quot;outputs\&quot;,             \&quot;path\&quot;: \&quot;/outputs\&quot;           }         ],         \&quot;Sharding\&quot;: {           \&quot;BatchSize\&quot;: 1,           \&quot;GlobPatternBasePath\&quot;: \&quot;/inputs\&quot;         }       },       \&quot;Deal\&quot;: {         \&quot;Concurrency\&quot;: 1       },       \&quot;ExecutionPlan\&quot;: {         \&quot;ShardsTotal\&quot;: 1       },       \&quot;CreatedAt\&quot;: \&quot;2022-11-17T13:32:55.33837275Z\&quot;,       \&quot;JobState\&quot;: {         \&quot;Nodes\&quot;: {           \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;: {             \&quot;Shards\&quot;: {               \&quot;0\&quot;: {                 \&quot;NodeId\&quot;: \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;,                 \&quot;State\&quot;: \&quot;Cancelled\&quot;,                 \&quot;VerificationResult\&quot;: {},                 \&quot;PublishedResults\&quot;: {}               }             }           },           \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;: {             \&quot;Shards\&quot;: {               \&quot;0\&quot;: {                 \&quot;NodeId\&quot;: \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;,                 \&quot;State\&quot;: \&quot;Cancelled\&quot;,                 \&quot;VerificationResult\&quot;: {},                 \&quot;PublishedResults\&quot;: {}               }             }           },           \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;: {             \&quot;Shards\&quot;: {               \&quot;0\&quot;: {                 \&quot;NodeId\&quot;: \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,                 \&quot;State\&quot;: \&quot;Completed\&quot;,                 \&quot;Status\&quot;: \&quot;Got results proposal of length: 0\&quot;,                 \&quot;VerificationResult\&quot;: {                   \&quot;Complete\&quot;: true,                   \&quot;Result\&quot;: true                 },                 \&quot;PublishedResults\&quot;: {                   \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,                   \&quot;Name\&quot;: \&quot;job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,                   \&quot;CID\&quot;: \&quot;QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\&quot;                 },                 \&quot;RunOutput\&quot;: {                   \&quot;stdout\&quot;: \&quot;Thu Nov 17 13:32:55 UTC 2022\\n\&quot;,                   \&quot;stdouttruncated\&quot;: false,                   \&quot;stderr\&quot;: \&quot;\&quot;,                   \&quot;stderrtruncated\&quot;: false,                   \&quot;exitCode\&quot;: 0,                   \&quot;runnerError\&quot;: \&quot;\&quot;                 }               }             }           }         }       }     },     {       \&quot;APIVersion\&quot;: \&quot;V1beta1\&quot;,       \&quot;ID\&quot;: \&quot;92d5d4ee-3765-4f78-8353-623f5f26df08\&quot;,       \&quot;RequesterNodeID\&quot;: \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,       \&quot;RequesterPublicKey\&quot;: \&quot;...\&quot;,       \&quot;ClientID\&quot;: \&quot;ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\&quot;,       \&quot;Spec\&quot;: {         \&quot;Engine\&quot;: \&quot;Docker\&quot;,         \&quot;Verifier\&quot;: \&quot;Noop\&quot;,         \&quot;Publisher\&quot;: \&quot;Estuary\&quot;,         \&quot;Docker\&quot;: {           \&quot;Image\&quot;: \&quot;ubuntu\&quot;,           \&quot;Entrypoint\&quot;: [             \&quot;sleep\&quot;,             \&quot;4\&quot;           ]         },         \&quot;Language\&quot;: {           \&quot;JobContext\&quot;: {}         },         \&quot;Wasm\&quot;: {},         \&quot;Resources\&quot;: {           \&quot;GPU\&quot;: \&quot;\&quot;         },         \&quot;Timeout\&quot;: 1800,         \&quot;outputs\&quot;: [           {             \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,             \&quot;Name\&quot;: \&quot;outputs\&quot;,             \&quot;path\&quot;: \&quot;/outputs\&quot;           }         ],         \&quot;Sharding\&quot;: {           \&quot;BatchSize\&quot;: 1,           \&quot;GlobPatternBasePath\&quot;: \&quot;/inputs\&quot;         }       },       \&quot;Deal\&quot;: {         \&quot;Concurrency\&quot;: 1       },       \&quot;ExecutionPlan\&quot;: {         \&quot;ShardsTotal\&quot;: 1       },       \&quot;CreatedAt\&quot;: \&quot;2022-11-17T13:29:01.871140291Z\&quot;,       \&quot;JobState\&quot;: {         \&quot;Nodes\&quot;: {           \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;: {             \&quot;Shards\&quot;: {               \&quot;0\&quot;: {                 \&quot;NodeId\&quot;: \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;,                 \&quot;State\&quot;: \&quot;Cancelled\&quot;,                 \&quot;VerificationResult\&quot;: {},                 \&quot;PublishedResults\&quot;: {}               }             }           },           \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;: {             \&quot;Shards\&quot;: {               \&quot;0\&quot;: {                 \&quot;NodeId\&quot;: \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;,                 \&quot;State\&quot;: \&quot;Completed\&quot;,                 \&quot;Status\&quot;: \&quot;Got results proposal of length: 0\&quot;,                 \&quot;VerificationResult\&quot;: {                   \&quot;Complete\&quot;: true,                   \&quot;Result\&quot;: true                 },                 \&quot;PublishedResults\&quot;: {                   \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,                   \&quot;Name\&quot;: \&quot;job-92d5d4ee-3765-4f78-8353-623f5f26df08-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;,                   \&quot;CID\&quot;: \&quot;QmWUXBndMuq2G6B6ndQCmkRHjZ6CvyJ8qLxXBG3YsSFzQG\&quot;                 },                 \&quot;RunOutput\&quot;: {                   \&quot;stdout\&quot;: \&quot;\&quot;,                   \&quot;stdouttruncated\&quot;: false,                   \&quot;stderr\&quot;: \&quot;\&quot;,                   \&quot;stderrtruncated\&quot;: false,                   \&quot;exitCode\&quot;: 0,                   \&quot;runnerError\&quot;: \&quot;\&quot;                 }               }             }           }         }       }     }   ] } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapilistRequest =  // PublicapilistRequest | Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!).
try {
    val result : PublicapilistResponse = apiInstance.pkg/publicapi.list(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/publicapi.list")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/publicapi.list")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapilistRequest**](PublicapilistRequest.md)| Set &#x60;return_all&#x60; to &#x60;true&#x60; to return all jobs on the network (may degrade performance, use with care!). |

### Return type

[**PublicapilistResponse**](PublicapilistResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

<a name="pkg/publicapi/events"></a>
# **pkg/publicapi/events**
> PublicapieventsResponse pkg/publicapi/events(body)

Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.

Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.  Example response (truncated): &#x60;&#x60;&#x60;json {   \&quot;events\&quot;: [     {       \&quot;APIVersion\&quot;: \&quot;V1beta1\&quot;,       \&quot;JobID\&quot;: \&quot;9304c616-291f-41ad-b862-54e133c0149e\&quot;,       \&quot;ClientID\&quot;: \&quot;ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\&quot;,       \&quot;SourceNodeID\&quot;: \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,       \&quot;EventName\&quot;: \&quot;Created\&quot;,       \&quot;Spec\&quot;: {         \&quot;Engine\&quot;: \&quot;Docker\&quot;,         \&quot;Verifier\&quot;: \&quot;Noop\&quot;,         \&quot;Publisher\&quot;: \&quot;Estuary\&quot;,         \&quot;Docker\&quot;: {           \&quot;Image\&quot;: \&quot;ubuntu\&quot;,           \&quot;Entrypoint\&quot;: [             \&quot;date\&quot;           ]         },         \&quot;Language\&quot;: {           \&quot;JobContext\&quot;: {}         },         \&quot;Wasm\&quot;: {},         \&quot;Resources\&quot;: {           \&quot;GPU\&quot;: \&quot;\&quot;         },         \&quot;Timeout\&quot;: 1800,         \&quot;outputs\&quot;: [           {             \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,             \&quot;Name\&quot;: \&quot;outputs\&quot;,             \&quot;path\&quot;: \&quot;/outputs\&quot;           }         ],         \&quot;Sharding\&quot;: {           \&quot;BatchSize\&quot;: 1,           \&quot;GlobPatternBasePath\&quot;: \&quot;/inputs\&quot;         }       },       \&quot;JobExecutionPlan\&quot;: {         \&quot;ShardsTotal\&quot;: 1       },       \&quot;Deal\&quot;: {         \&quot;Concurrency\&quot;: 1       },       \&quot;VerificationResult\&quot;: {},       \&quot;PublishedResult\&quot;: {},       \&quot;EventTime\&quot;: \&quot;2022-11-17T13:32:55.331375351Z\&quot;,       \&quot;SenderPublicKey\&quot;: \&quot;...\&quot;     },     ...     {       \&quot;JobID\&quot;: \&quot;9304c616-291f-41ad-b862-54e133c0149e\&quot;,       \&quot;SourceNodeID\&quot;: \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,       \&quot;TargetNodeID\&quot;: \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,       \&quot;EventName\&quot;: \&quot;ResultsAccepted\&quot;,       \&quot;Spec\&quot;: {         \&quot;Docker\&quot;: {},         \&quot;Language\&quot;: {           \&quot;JobContext\&quot;: {}         },         \&quot;Wasm\&quot;: {},         \&quot;Resources\&quot;: {           \&quot;GPU\&quot;: \&quot;\&quot;         },         \&quot;Sharding\&quot;: {}       },       \&quot;JobExecutionPlan\&quot;: {},       \&quot;Deal\&quot;: {},       \&quot;VerificationResult\&quot;: {         \&quot;Complete\&quot;: true,         \&quot;Result\&quot;: true       },       \&quot;PublishedResult\&quot;: {},       \&quot;EventTime\&quot;: \&quot;2022-11-17T13:32:55.707825569Z\&quot;,       \&quot;SenderPublicKey\&quot;: \&quot;...\&quot;     },     {       \&quot;JobID\&quot;: \&quot;9304c616-291f-41ad-b862-54e133c0149e\&quot;,       \&quot;SourceNodeID\&quot;: \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,       \&quot;EventName\&quot;: \&quot;ResultsPublished\&quot;,       \&quot;Spec\&quot;: {         \&quot;Docker\&quot;: {},         \&quot;Language\&quot;: {           \&quot;JobContext\&quot;: {}         },         \&quot;Wasm\&quot;: {},         \&quot;Resources\&quot;: {           \&quot;GPU\&quot;: \&quot;\&quot;         },         \&quot;Sharding\&quot;: {}       },       \&quot;JobExecutionPlan\&quot;: {},       \&quot;Deal\&quot;: {},       \&quot;VerificationResult\&quot;: {},       \&quot;PublishedResult\&quot;: {         \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,         \&quot;Name\&quot;: \&quot;job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,         \&quot;CID\&quot;: \&quot;QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\&quot;       },       \&quot;EventTime\&quot;: \&quot;2022-11-17T13:32:55.756658941Z\&quot;,       \&quot;SenderPublicKey\&quot;: \&quot;...\&quot;     }   ] } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapieventsRequest =  // PublicapieventsRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
try {
    val result : PublicapieventsResponse = apiInstance.pkg/publicapi/events(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/publicapi/events")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/publicapi/events")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapieventsRequest**](PublicapieventsRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. |

### Return type

[**PublicapieventsResponse**](PublicapieventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

<a name="pkg/publicapi/localEvents"></a>
# **pkg/publicapi/localEvents**
> PublicapilocalEventsResponse pkg/publicapi/localEvents(body)

Returns the node&#x27;s local events related to the job-id passed in the body payload. Useful for troubleshooting.

Local events (e.g. Selected, BidAccepted, Verified) are useful to track the progress of a job.

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapilocalEventsRequest =  // PublicapilocalEventsRequest | 
try {
    val result : PublicapilocalEventsResponse = apiInstance.pkg/publicapi/localEvents(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/publicapi/localEvents")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/publicapi/localEvents")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapilocalEventsRequest**](PublicapilocalEventsRequest.md)|  |

### Return type

[**PublicapilocalEventsResponse**](PublicapilocalEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

<a name="pkg/publicapi/results"></a>
# **pkg/publicapi/results**
> PublicapiresultsResponse pkg/publicapi/results(body)

Returns the results of the job-id specified in the body payload.

Example response:  &#x60;&#x60;&#x60;json {   \&quot;results\&quot;: [     {       \&quot;NodeID\&quot;: \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,       \&quot;Data\&quot;: {         \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,         \&quot;Name\&quot;: \&quot;job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,         \&quot;CID\&quot;: \&quot;QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\&quot;       }     }   ] } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapistateRequest =  // PublicapistateRequest | 
try {
    val result : PublicapiresultsResponse = apiInstance.pkg/publicapi/results(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/publicapi/results")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/publicapi/results")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapistateRequest**](PublicapistateRequest.md)|  |

### Return type

[**PublicapiresultsResponse**](PublicapiresultsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

<a name="pkg/publicapi/states"></a>
# **pkg/publicapi/states**
> PublicapistateResponse pkg/publicapi/states(body)

Returns the state of the job-id specified in the body payload.

Example response:  &#x60;&#x60;&#x60;json {   \&quot;state\&quot;: {     \&quot;Nodes\&quot;: {       \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;: {         \&quot;Shards\&quot;: {           \&quot;0\&quot;: {             \&quot;NodeId\&quot;: \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;,             \&quot;State\&quot;: \&quot;Cancelled\&quot;,             \&quot;VerificationResult\&quot;: {},             \&quot;PublishedResults\&quot;: {}           }         }       },       \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;: {         \&quot;Shards\&quot;: {           \&quot;0\&quot;: {             \&quot;NodeId\&quot;: \&quot;QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\&quot;,             \&quot;State\&quot;: \&quot;Cancelled\&quot;,             \&quot;VerificationResult\&quot;: {},             \&quot;PublishedResults\&quot;: {}           }         }       },       \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;: {         \&quot;Shards\&quot;: {           \&quot;0\&quot;: {             \&quot;NodeId\&quot;: \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,             \&quot;State\&quot;: \&quot;Completed\&quot;,             \&quot;Status\&quot;: \&quot;Got results proposal of length: 0\&quot;,             \&quot;VerificationResult\&quot;: {               \&quot;Complete\&quot;: true,               \&quot;Result\&quot;: true             },             \&quot;PublishedResults\&quot;: {               \&quot;StorageSource\&quot;: \&quot;IPFS\&quot;,               \&quot;Name\&quot;: \&quot;job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,               \&quot;CID\&quot;: \&quot;QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\&quot;             },             \&quot;RunOutput\&quot;: {               \&quot;stdout\&quot;: \&quot;Thu Nov 17 13:32:55 UTC 2022\\n\&quot;,               \&quot;stdouttruncated\&quot;: false,               \&quot;stderr\&quot;: \&quot;\&quot;,               \&quot;stderrtruncated\&quot;: false,               \&quot;exitCode\&quot;: 0,               \&quot;runnerError\&quot;: \&quot;\&quot;             }           }         }       }     }   } } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = JobApi()
val body : PublicapistateRequest =  // PublicapistateRequest | 
try {
    val result : PublicapistateResponse = apiInstance.pkg/publicapi/states(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling JobApi#pkg/publicapi/states")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling JobApi#pkg/publicapi/states")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapistateRequest**](PublicapistateRequest.md)|  |

### Return type

[**PublicapistateResponse**](PublicapistateResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

