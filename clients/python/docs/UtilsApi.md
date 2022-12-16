# bacalhau_apiclient.UtilsApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234*

Method | HTTP request | Description
------------- | ------------- | -------------
[**debug**](UtilsApi.md#debug) | **GET** /debug | Returns debug information on what the current node is doing.
[**healthz**](UtilsApi.md#healthz) | **GET** /healthz | 
[**id**](UtilsApi.md#id) | **GET** /id | Returns the id of the host node.
[**livez**](UtilsApi.md#livez) | **GET** /livez | 
[**logz**](UtilsApi.md#logz) | **GET** /logz | 
[**peers**](UtilsApi.md#peers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**readyz**](UtilsApi.md#readyz) | **GET** /readyz | 
[**varz**](UtilsApi.md#varz) | **GET** /varz | 
[**version**](UtilsApi.md#version) | **POST** /version | Returns the build version running on the server.


# **debug**
> str debug()

Returns debug information on what the current node is doing.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    # Returns debug information on what the current node is doing.
    api_response = api_instance.debug()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->debug: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

**str**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **healthz**
> HealthInfo healthz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    api_response = api_instance.healthz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->healthz: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**HealthInfo**](HealthInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **id**
> str id()

Returns the id of the host node.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    # Returns the id of the host node.
    api_response = api_instance.id()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->id: %s\n" % e)
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

# **livez**
> str livez()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    api_response = api_instance.livez()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->livez: %s\n" % e)
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

# **logz**
> str logz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    api_response = api_instance.logz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->logz: %s\n" % e)
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

# **peers**
> dict(str, list[str]) peers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    # Returns the peers connected to the host via the transport layer.
    api_response = api_instance.peers()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->peers: %s\n" % e)
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

# **readyz**
> str readyz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    api_response = api_instance.readyz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->readyz: %s\n" % e)
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

# **varz**
> list[int] varz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.UtilsApi()

try:
    api_response = api_instance.varz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->varz: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

**list[int]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **version**
> VersionResponse version(version_request)

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
api_instance = bacalhau_apiclient.UtilsApi()
version_request = bacalhau_apiclient.VersionRequest() # VersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try:
    # Returns the build version running on the server.
    api_response = api_instance.version(version_request)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->version: %s\n" % e)
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

