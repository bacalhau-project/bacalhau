# bacalhau_apiclient.JobApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234*

Method | HTTP request | Description
------------- | ------------- | -------------
[**events**](JobApi.md#events) | **POST** /events | Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
[**list**](JobApi.md#list) | **POST** /list | Simply lists jobs.
[**local_events**](JobApi.md#local_events) | **POST** /local_events | Returns the node&#39;s local events related to the job-id passed in the body payload. Useful for troubleshooting.
[**results**](JobApi.md#results) | **POST** /results | Returns the results of the job-id specified in the body payload.
[**states**](JobApi.md#states) | **POST** /states | Returns the state of the job-id specified in the body payload.
[**submit**](JobApi.md#submit) | **POST** /submit | Submits a new job to the network.


# **events**
> EventsResponse events(events_request)

Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.

Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.  Example response (truncated): ```json {   \"events\": [     {       \"APIVersion\": \"V1beta1\",       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"EventName\": \"Created\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"JobExecutionPlan\": {         \"ShardsTotal\": 1       },       \"Deal\": {         \"Concurrency\": 1       },       \"VerificationResult\": {},       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.331375351Z\",       \"SenderPublicKey\": \"...\"     },     ...     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"TargetNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsAccepted\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {         \"Complete\": true,         \"Result\": true       },       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.707825569Z\",       \"SenderPublicKey\": \"...\"     },     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsPublished\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {},       \"PublishedResult\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       },       \"EventTime\": \"2022-11-17T13:32:55.756658941Z\",       \"SenderPublicKey\": \"...\"     }   ] } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
events_request = bacalhau_apiclient.EventsRequest() # EventsRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try:
    # Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
    api_response = api_instance.events(events_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->events: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **events_request** | [**EventsRequest**](EventsRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. | 

### Return type

[**EventsResponse**](EventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list**
> ListResponse list(list_request)

Simply lists jobs.

Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default). If `return_all` is set to true, it returns all jobs on the Bacalhau network.  If `id` is set, it returns only the job with that ID.  Example response: ```json {   \"jobs\": [     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:32:55.33837275Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                   \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"                 },                 \"RunOutput\": {                   \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     },     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"92d5d4ee-3765-4f78-8353-623f5f26df08\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"sleep\",             \"4\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:29:01.871140291Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-92d5d4ee-3765-4f78-8353-623f5f26df08-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                   \"CID\": \"QmWUXBndMuq2G6B6ndQCmkRHjZ6CvyJ8qLxXBG3YsSFzQG\"                 },                 \"RunOutput\": {                   \"stdout\": \"\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     }   ] } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
list_request = bacalhau_apiclient.ListRequest() # ListRequest | Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!).

try:
    # Simply lists jobs.
    api_response = api_instance.list(list_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->list: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **list_request** | [**ListRequest**](ListRequest.md)| Set &#x60;return_all&#x60; to &#x60;true&#x60; to return all jobs on the network (may degrade performance, use with care!). | 

### Return type

[**ListResponse**](ListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **local_events**
> LocalEventsResponse local_events(local_events_request)

Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.

Local events (e.g. Selected, BidAccepted, Verified) are useful to track the progress of a job.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
local_events_request = bacalhau_apiclient.LocalEventsRequest() # LocalEventsRequest |  

try:
    # Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
    api_response = api_instance.local_events(local_events_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->local_events: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **local_events_request** | [**LocalEventsRequest**](LocalEventsRequest.md)|   | 

### Return type

[**LocalEventsResponse**](LocalEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **results**
> ResultsResponse results(state_request)

Returns the results of the job-id specified in the body payload.

Example response:  ```json {   \"results\": [     {       \"NodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"Data\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       }     }   ] } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
state_request = bacalhau_apiclient.StateRequest() # StateRequest |  

try:
    # Returns the results of the job-id specified in the body payload.
    api_response = api_instance.results(state_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->results: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **state_request** | [**StateRequest**](StateRequest.md)|   | 

### Return type

[**ResultsResponse**](ResultsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **states**
> StateResponse states(state_request)

Returns the state of the job-id specified in the body payload.

Example response:  ```json {   \"state\": {     \"Nodes\": {       \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",             \"State\": \"Completed\",             \"Status\": \"Got results proposal of length: 0\",             \"VerificationResult\": {               \"Complete\": true,               \"Result\": true             },             \"PublishedResults\": {               \"StorageSource\": \"IPFS\",               \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",               \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"             },             \"RunOutput\": {               \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",               \"stdouttruncated\": false,               \"stderr\": \"\",               \"stderrtruncated\": false,               \"exitCode\": 0,               \"runnerError\": \"\"             }           }         }       }     }   } } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
state_request = bacalhau_apiclient.StateRequest() # StateRequest |  

try:
    # Returns the state of the job-id specified in the body payload.
    api_response = api_instance.states(state_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->states: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **state_request** | [**StateRequest**](StateRequest.md)|   | 

### Return type

[**StateResponse**](StateResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **submit**
> SubmitResponse submit(submit_request)

Submits a new job to the network.

Description:  * `client_public_key`: The base64-encoded public key of the client. * `signature`: A base64-encoded signature of the `data` attribute, signed by the client. * `job_create_payload`:     * `ClientID`: Request must specify a `ClientID`. To retrieve your `ClientID`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.  * `APIVersion`: e.g. `\"V1beta1\"`.     * `Spec`: https://github.com/filecoin-project/bacalhau/blob/main/pkg/job.go 

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
submit_request = bacalhau_apiclient.SubmitRequest() # SubmitRequest |  

try:
    # Submits a new job to the network.
    api_response = api_instance.submit(submit_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->submit: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **submit_request** | [**SubmitRequest**](SubmitRequest.md)|   | 

### Return type

[**SubmitResponse**](SubmitResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

