# bacalhau_client.MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_serverid**](MiscApi.md#api_serverid) | **GET** /id | Returns the id of the host node.
[**api_serverpeers**](MiscApi.md#api_serverpeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**api_serverversion**](MiscApi.md#api_serverversion) | **POST** /version | Returns the build version running on the server.

# **api_serverid**
> str api_serverid()

Returns the id of the host node.

### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.MiscApi()

try:
    # Returns the id of the host node.
    api_response = api_instance.api_serverid()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling MiscApi->api_serverid: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

**str**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_serverpeers**
> dict(str, list[str]) api_serverpeers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.MiscApi()

try:
    # Returns the peers connected to the host via the transport layer.
    api_response = api_instance.api_serverpeers()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling MiscApi->api_serverpeers: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

**dict(str, list[str])**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_serverversion**
> PublicapiVersionResponse api_serverversion(body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.MiscApi()
body = bacalhau_client.PublicapiVersionRequest() # PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try:
    # Returns the build version running on the server.
    api_response = api_instance.api_serverversion(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling MiscApi->api_serverversion: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiVersionRequest**](PublicapiVersionRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. | 

### Return type

[**PublicapiVersionResponse**](PublicapiVersionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

