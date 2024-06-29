# bacalhau_apiclient.UtilsApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**healthz**](UtilsApi.md#healthz) | **GET** /api/v1/healthz | 
[**home**](UtilsApi.md#home) | **GET** / | 
[**id**](UtilsApi.md#id) | **GET** /api/v1/id | Returns the id of the host node.
[**livez**](UtilsApi.md#livez) | **GET** /api/v1/livez | 
[**node_info**](UtilsApi.md#node_info) | **GET** /api/v1/node_info | Returns the info of the node.

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

# **home**
> str home()



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
    api_response = api_instance.home()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UtilsApi->home: %s\n" % e)
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

