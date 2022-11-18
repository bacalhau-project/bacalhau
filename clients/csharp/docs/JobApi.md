# bacalhau-client.Api.JobApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**PkgapiServerSubmit**](JobApi.md#pkgapiserversubmit) | **POST** /submit | Submits a new job to the network.
[**PkgpublicapiList**](JobApi.md#pkgpublicapilist) | **POST** /list | Simply lists jobs.
[**Pkgpublicapievents**](JobApi.md#pkgpublicapievents) | **POST** /events | Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
[**PkgpublicapilocalEvents**](JobApi.md#pkgpublicapilocalevents) | **POST** /local_events | Returns the node&#x27;s local events related to the job-id passed in the body payload. Useful for troubleshooting.
[**Pkgpublicapiresults**](JobApi.md#pkgpublicapiresults) | **POST** /results | Returns the results of the job-id specified in the body payload.
[**Pkgpublicapistates**](JobApi.md#pkgpublicapistates) | **POST** /states | Returns the state of the job-id specified in the body payload.

<a name="pkgapiserversubmit"></a>
# **PkgapiServerSubmit**
> PublicapiSubmitResponse PkgapiServerSubmit (PublicapiSubmitRequest body)

Submits a new job to the network.

Description:  * `client_public_key`: The base64-encoded public key of the client. * `signature`: A base64-encoded signature of the `data` attribute, signed by the client. * `data`     * `ClientID`: Request must specify a `ClientID`. To retrieve your `ClientID`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.     * `Job`: see example below.  Example request ```json {  \"data\": {   \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",   \"Job\": {    \"APIVersion\": \"V1beta1\",    \"Spec\": {     \"Engine\": \"Docker\",     \"Verifier\": \"Noop\",     \"Publisher\": \"Estuary\",     \"Docker\": {      \"Image\": \"ubuntu\",      \"Entrypoint\": [       \"date\"      ]     },     \"Timeout\": 1800,     \"outputs\": [      {       \"StorageSource\": \"IPFS\",       \"Name\": \"outputs\",       \"path\": \"/outputs\"      }     ],     \"Sharding\": {      \"BatchSize\": 1,      \"GlobPatternBasePath\": \"/inputs\"     }    },    \"Deal\": {     \"Concurrency\": 1    }   }  },  \"signature\": \"...\",  \"client_public_key\": \"...\" } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgapiServerSubmitExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiSubmitRequest(); // PublicapiSubmitRequest | 

            try
            {
                // Submits a new job to the network.
                PublicapiSubmitResponse result = apiInstance.PkgapiServerSubmit(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.PkgapiServerSubmit: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiSubmitRequest**](PublicapiSubmitRequest.md)|  | 

### Return type

[**PublicapiSubmitResponse**](PublicapiSubmitResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="pkgpublicapilist"></a>
# **PkgpublicapiList**
> PublicapiListResponse PkgpublicapiList (PublicapiListRequest body)

Simply lists jobs.

Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default). If `return_all` is set to true, it returns all jobs on the Bacalhau network.  If `id` is set, it returns only the job with that ID.  Example response: ```json {   \"jobs\": [     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:32:55.33837275Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",                   \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"                 },                 \"RunOutput\": {                   \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     },     {       \"APIVersion\": \"V1beta1\",       \"ID\": \"92d5d4ee-3765-4f78-8353-623f5f26df08\",       \"RequesterNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"RequesterPublicKey\": \"...\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"sleep\",             \"4\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"Deal\": {         \"Concurrency\": 1       },       \"ExecutionPlan\": {         \"ShardsTotal\": 1       },       \"CreatedAt\": \"2022-11-17T13:29:01.871140291Z\",       \"JobState\": {         \"Nodes\": {           \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",                 \"State\": \"Cancelled\",                 \"VerificationResult\": {},                 \"PublishedResults\": {}               }             }           },           \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {             \"Shards\": {               \"0\": {                 \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                 \"State\": \"Completed\",                 \"Status\": \"Got results proposal of length: 0\",                 \"VerificationResult\": {                   \"Complete\": true,                   \"Result\": true                 },                 \"PublishedResults\": {                   \"StorageSource\": \"IPFS\",                   \"Name\": \"job-92d5d4ee-3765-4f78-8353-623f5f26df08-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",                   \"CID\": \"QmWUXBndMuq2G6B6ndQCmkRHjZ6CvyJ8qLxXBG3YsSFzQG\"                 },                 \"RunOutput\": {                   \"stdout\": \"\",                   \"stdouttruncated\": false,                   \"stderr\": \"\",                   \"stderrtruncated\": false,                   \"exitCode\": 0,                   \"runnerError\": \"\"                 }               }             }           }         }       }     }   ] } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgpublicapiListExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiListRequest(); // PublicapiListRequest | Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!).

            try
            {
                // Simply lists jobs.
                PublicapiListResponse result = apiInstance.PkgpublicapiList(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.PkgpublicapiList: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiListRequest**](PublicapiListRequest.md)| Set &#x60;return_all&#x60; to &#x60;true&#x60; to return all jobs on the network (may degrade performance, use with care!). | 

### Return type

[**PublicapiListResponse**](PublicapiListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="pkgpublicapievents"></a>
# **Pkgpublicapievents**
> PublicapiEventsResponse Pkgpublicapievents (PublicapiEventsRequest body)

Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.

Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.  Example response (truncated): ```json {   \"events\": [     {       \"APIVersion\": \"V1beta1\",       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"ClientID\": \"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"EventName\": \"Created\",       \"Spec\": {         \"Engine\": \"Docker\",         \"Verifier\": \"Noop\",         \"Publisher\": \"Estuary\",         \"Docker\": {           \"Image\": \"ubuntu\",           \"Entrypoint\": [             \"date\"           ]         },         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Timeout\": 1800,         \"outputs\": [           {             \"StorageSource\": \"IPFS\",             \"Name\": \"outputs\",             \"path\": \"/outputs\"           }         ],         \"Sharding\": {           \"BatchSize\": 1,           \"GlobPatternBasePath\": \"/inputs\"         }       },       \"JobExecutionPlan\": {         \"ShardsTotal\": 1       },       \"Deal\": {         \"Concurrency\": 1       },       \"VerificationResult\": {},       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.331375351Z\",       \"SenderPublicKey\": \"...\"     },     ...     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",       \"TargetNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsAccepted\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {         \"Complete\": true,         \"Result\": true       },       \"PublishedResult\": {},       \"EventTime\": \"2022-11-17T13:32:55.707825569Z\",       \"SenderPublicKey\": \"...\"     },     {       \"JobID\": \"9304c616-291f-41ad-b862-54e133c0149e\",       \"SourceNodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"EventName\": \"ResultsPublished\",       \"Spec\": {         \"Docker\": {},         \"Language\": {           \"JobContext\": {}         },         \"Wasm\": {},         \"Resources\": {           \"GPU\": \"\"         },         \"Sharding\": {}       },       \"JobExecutionPlan\": {},       \"Deal\": {},       \"VerificationResult\": {},       \"PublishedResult\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       },       \"EventTime\": \"2022-11-17T13:32:55.756658941Z\",       \"SenderPublicKey\": \"...\"     }   ] } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgpublicapieventsExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiEventsRequest(); // PublicapiEventsRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

            try
            {
                // Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
                PublicapiEventsResponse result = apiInstance.Pkgpublicapievents(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.Pkgpublicapievents: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiEventsRequest**](PublicapiEventsRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. | 

### Return type

[**PublicapiEventsResponse**](PublicapiEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="pkgpublicapilocalevents"></a>
# **PkgpublicapilocalEvents**
> PublicapiLocalEventsResponse PkgpublicapilocalEvents (PublicapiLocalEventsRequest body)

Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.

Local events (e.g. Selected, BidAccepted, Verified) are useful to track the progress of a job.

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgpublicapilocalEventsExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiLocalEventsRequest(); // PublicapiLocalEventsRequest | 

            try
            {
                // Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
                PublicapiLocalEventsResponse result = apiInstance.PkgpublicapilocalEvents(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.PkgpublicapilocalEvents: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiLocalEventsRequest**](PublicapiLocalEventsRequest.md)|  | 

### Return type

[**PublicapiLocalEventsResponse**](PublicapiLocalEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="pkgpublicapiresults"></a>
# **Pkgpublicapiresults**
> PublicapiResultsResponse Pkgpublicapiresults (PublicapiStateRequest body)

Returns the results of the job-id specified in the body payload.

Example response:  ```json {   \"results\": [     {       \"NodeID\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",       \"Data\": {         \"StorageSource\": \"IPFS\",         \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",         \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"       }     }   ] } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgpublicapiresultsExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiStateRequest(); // PublicapiStateRequest | 

            try
            {
                // Returns the results of the job-id specified in the body payload.
                PublicapiResultsResponse result = apiInstance.Pkgpublicapiresults(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.Pkgpublicapiresults: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiStateRequest**](PublicapiStateRequest.md)|  | 

### Return type

[**PublicapiResultsResponse**](PublicapiResultsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="pkgpublicapistates"></a>
# **Pkgpublicapistates**
> PublicapiStateResponse Pkgpublicapistates (PublicapiStateRequest body)

Returns the state of the job-id specified in the body payload.

Example response:  ```json {   \"state\": {     \"Nodes\": {       \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3\",             \"State\": \"Cancelled\",             \"VerificationResult\": {},             \"PublishedResults\": {}           }         }       },       \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\": {         \"Shards\": {           \"0\": {             \"NodeId\": \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",             \"State\": \"Completed\",             \"Status\": \"Got results proposal of length: 0\",             \"VerificationResult\": {               \"Complete\": true,               \"Result\": true             },             \"PublishedResults\": {               \"StorageSource\": \"IPFS\",               \"Name\": \"job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",               \"CID\": \"QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe\"             },             \"RunOutput\": {               \"stdout\": \"Thu Nov 17 13:32:55 UTC 2022\\n\",               \"stdouttruncated\": false,               \"stderr\": \"\",               \"stderrtruncated\": false,               \"exitCode\": 0,               \"runnerError\": \"\"             }           }         }       }     }   } } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class PkgpublicapistatesExample
    {
        public void main()
        {
            var apiInstance = new JobApi();
            var body = new PublicapiStateRequest(); // PublicapiStateRequest | 

            try
            {
                // Returns the state of the job-id specified in the body payload.
                PublicapiStateResponse result = apiInstance.Pkgpublicapistates(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling JobApi.Pkgpublicapistates: " + e.Message );
            }
        }
    }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiStateRequest**](PublicapiStateRequest.md)|  | 

### Return type

[**PublicapiStateResponse**](PublicapiStateResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
