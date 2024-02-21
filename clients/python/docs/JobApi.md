# bacalhau_apiclient.JobApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**cancel**](JobApi.md#cancel) | **POST** /api/v1/requester/cancel | Cancels the job with the job-id specified in the body payload.
[**events**](JobApi.md#events) | **POST** /api/v1/requester/events | Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
[**list**](JobApi.md#list) | **POST** /api/v1/requester/list | Simply lists jobs.
[**logs**](JobApi.md#logs) | **POST** /api/v1/requester/logs | Displays the logs for a current job/execution
[**results**](JobApi.md#results) | **POST** /api/v1/requester/results | Returns the results of the job-id specified in the body payload.
[**states**](JobApi.md#states) | **POST** /api/v1/requester/states | Returns the state of the job-id specified in the body payload.
[**submit**](JobApi.md#submit) | **POST** /api/v1/requester/submit | Submits a new job to the network.

# **cancel**
> LegacyCancelResponse cancel(body)

Cancels the job with the job-id specified in the body payload.

Cancels a job specified by `id` as long as that job belongs to `client_id`. Returns the current jobstate after the cancel request has been processed.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyCancelRequest() # LegacyCancelRequest |

try:
    # Cancels the job with the job-id specified in the body payload.
    api_response = api_instance.cancel(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->cancel: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyCancelRequest**](LegacyCancelRequest.md)|  |

### Return type

[**LegacyCancelResponse**](LegacyCancelResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **events**
> LegacyEventsResponse events(body)

Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.

Events (e.g. Created, Bid, BidAccepted, ..., ResultsAccepted, ResultsPublished) are useful to track the progress of a job.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyEventsRequest() # LegacyEventsRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try:
    # Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
    api_response = api_instance.events(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->events: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyEventsRequest**](LegacyEventsRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. |

### Return type

[**LegacyEventsResponse**](LegacyEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list**
> LegacyListResponse list(body)

Simply lists jobs.

Returns the first (sorted) #`max_jobs` jobs that belong to the `client_id` passed in the body payload (by default). If `return_all` is set to true, it returns all jobs on the Bacalhau network. If `id` is set, it returns only the job with that ID.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyListRequest() # LegacyListRequest | Set `return_all` to `true` to return all jobs on the network (may degrade performance, use with care!).

try:
    # Simply lists jobs.
    api_response = api_instance.list(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->list: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyListRequest**](LegacyListRequest.md)| Set &#x60;return_all&#x60; to &#x60;true&#x60; to return all jobs on the network (may degrade performance, use with care!). |

### Return type

[**LegacyListResponse**](LegacyListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **logs**
> str logs(body)

Displays the logs for a current job/execution

Shows the output from the job specified by `id` as long as that job belongs to `client_id`. The output will be continuous until either, the client disconnects or the execution completes.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyLogRequest() # LegacyLogRequest |

try:
    # Displays the logs for a current job/execution
    api_response = api_instance.logs(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->logs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyLogRequest**](LegacyLogRequest.md)|  |

### Return type

**str**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **results**
> LegacyResultsResponse results(body)

Returns the results of the job-id specified in the body payload.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyStateRequest() # LegacyStateRequest |

try:
    # Returns the results of the job-id specified in the body payload.
    api_response = api_instance.results(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->results: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyStateRequest**](LegacyStateRequest.md)|  |

### Return type

[**LegacyResultsResponse**](LegacyResultsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **states**
> LegacyStateResponse states(body)

Returns the state of the job-id specified in the body payload.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacyStateRequest() # LegacyStateRequest |

try:
    # Returns the state of the job-id specified in the body payload.
    api_response = api_instance.states(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->states: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacyStateRequest**](LegacyStateRequest.md)|  |

### Return type

[**LegacyStateResponse**](LegacyStateResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **submit**
> LegacySubmitResponse submit(body)

Submits a new job to the network.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.JobApi()
body = bacalhau_apiclient.LegacySubmitRequest() # LegacySubmitRequest |

try:
    # Submits a new job to the network.
    api_response = api_instance.submit(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling JobApi->submit: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**LegacySubmitRequest**](LegacySubmitRequest.md)|  |

### Return type

[**LegacySubmitResponse**](LegacySubmitResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
