# bacalhau_apiclient.OpsApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**agentalive**](OpsApi.md#agentalive) | **GET** /api/v1/agent/alive | 
[**agentdebug**](OpsApi.md#agentdebug) | **GET** /api/v1/agent/debug | Returns debug information on what the current node is doing.
[**agentnode**](OpsApi.md#agentnode) | **GET** /api/v1/agent/node | Returns the info of the node.
[**agentversion**](OpsApi.md#agentversion) | **GET** /api/v1/agent/version | Returns the build version running on the server.

# **agentalive**
> str agentalive()



### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.OpsApi()

try:
    api_response = api_instance.agentalive()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling OpsApi->agentalive: %s\n" % e)
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

# **agentdebug**
> DebugInfo agentdebug()

Returns debug information on what the current node is doing.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.OpsApi()

try:
    # Returns debug information on what the current node is doing.
    api_response = api_instance.agentdebug()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling OpsApi->agentdebug: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**DebugInfo**](DebugInfo.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **agentnode**
> NodeInfo agentnode()

Returns the info of the node.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.OpsApi()

try:
    # Returns the info of the node.
    api_response = api_instance.agentnode()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling OpsApi->agentnode: %s\n" % e)
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

# **agentversion**
> ApiGetVersionResponse agentversion()

Returns the build version running on the server.

See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```python
from __future__ import print_function
import time
import bacalhau_apiclient
from bacalhau_apiclient.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = bacalhau_apiclient.OpsApi()

try:
    # Returns the build version running on the server.
    api_response = api_instance.agentversion()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling OpsApi->agentversion: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ApiGetVersionResponse**](ApiGetVersionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

