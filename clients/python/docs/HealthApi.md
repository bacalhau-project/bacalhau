# bacalhau_client.HealthApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_serverdebug**](HealthApi.md#api_serverdebug) | **GET** /debug | Returns debug information on what the current node is doing.
[**api_serverhealthz**](HealthApi.md#api_serverhealthz) | **GET** /healthz | 
[**api_serverlivez**](HealthApi.md#api_serverlivez) | **GET** /livez | 
[**api_serverlogz**](HealthApi.md#api_serverlogz) | **GET** /logz | 
[**api_serverreadyz**](HealthApi.md#api_serverreadyz) | **GET** /readyz | 
[**api_servervarz**](HealthApi.md#api_servervarz) | **GET** /varz | 

# **api_serverdebug**
> PublicapiDebugResponse api_serverdebug()

Returns debug information on what the current node is doing.

### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    # Returns debug information on what the current node is doing.
    api_response = api_instance.api_serverdebug()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_serverdebug: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**PublicapiDebugResponse**](PublicapiDebugResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_serverhealthz**
> TypesHealthInfo api_serverhealthz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    api_response = api_instance.api_serverhealthz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_serverhealthz: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TypesHealthInfo**](TypesHealthInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_serverlivez**
> str api_serverlivez()



### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    api_response = api_instance.api_serverlivez()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_serverlivez: %s\n" % e)
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

# **api_serverlogz**
> str api_serverlogz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    api_response = api_instance.api_serverlogz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_serverlogz: %s\n" % e)
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

# **api_serverreadyz**
> str api_serverreadyz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    api_response = api_instance.api_serverreadyz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_serverreadyz: %s\n" % e)
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

# **api_servervarz**
> list[int] api_servervarz()



### Example
```python
from __future__ import print_function
import time
import bacalhau_client
from bacalhau_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_client.HealthApi()

try:
    api_response = api_instance.api_servervarz()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling HealthApi->api_servervarz: %s\n" % e)
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

