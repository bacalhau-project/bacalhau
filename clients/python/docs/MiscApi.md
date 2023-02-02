# bacalhau_apiclient.MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_serverversion**](MiscApi.md#api_serverversion) | **POST** /version | Returns the build version running on the server.


# **api_serverversion**
> VersionResponse api_serverversion(version_request)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.MiscApi()
version_request = bacalhau_apiclient.VersionRequest() # VersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try:
    # Returns the build version running on the server.
    api_response = api_instance.api_serverversion(version_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling MiscApi->api_serverversion: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **version_request** | [**VersionRequest**](VersionRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. | 

### Return type

[**VersionResponse**](VersionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

