# bacalhau_apiclient.UtilsApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234*

Method | HTTP request | Description
------------- | ------------- | -------------
[**healthz**](UtilsApi.md#healthz) | **GET** /healthz | 
[**id**](UtilsApi.md#id) | **GET** /id | Returns the id of the host node.
[**livez**](UtilsApi.md#livez) | **GET** /livez | 
[**logz**](UtilsApi.md#logz) | **GET** /logz | 
[**node_info**](UtilsApi.md#node_info) | **GET** /node_info | Returns the info of the node.
[**peers**](UtilsApi.md#peers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**readyz**](UtilsApi.md#readyz) | **GET** /readyz | 
[**varz**](UtilsApi.md#varz) | **GET** /varz | 


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

# **node_info**
> NodeInfo node_info()

Returns the info of the node.

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
    # Returns the info of the node.
    api_response = api_instance.node_info()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->node_info: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**NodeInfo**](NodeInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **peers**
> list[PeerAddrInfo] peers()

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

[**list[PeerAddrInfo]**](PeerAddrInfo.md)

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

