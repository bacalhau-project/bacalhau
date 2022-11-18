# BacalhauClient.HealthApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServerdebug**](HealthApi.md#apiServerdebug) | **GET** /debug | Returns debug information on what the current node is doing.
[**apiServerhealthz**](HealthApi.md#apiServerhealthz) | **GET** /healthz | 
[**apiServerlivez**](HealthApi.md#apiServerlivez) | **GET** /livez | 
[**apiServerlogz**](HealthApi.md#apiServerlogz) | **GET** /logz | 
[**apiServerreadyz**](HealthApi.md#apiServerreadyz) | **GET** /readyz | 
[**apiServervarz**](HealthApi.md#apiServervarz) | **GET** /varz | 

<a name="apiServerdebug"></a>
# **apiServerdebug**
> PublicapiDebugResponse apiServerdebug()

Returns debug information on what the current node is doing.

### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServerdebug((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
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

<a name="apiServerhealthz"></a>
# **apiServerhealthz**
> TypesHealthInfo apiServerhealthz()



### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServerhealthz((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
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

<a name="apiServerlivez"></a>
# **apiServerlivez**
> &#x27;String&#x27; apiServerlivez()



### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServerlivez((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
```

### Parameters
This endpoint does not need any parameter.

### Return type

**&#x27;String&#x27;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServerlogz"></a>
# **apiServerlogz**
> &#x27;String&#x27; apiServerlogz()



### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServerlogz((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
```

### Parameters
This endpoint does not need any parameter.

### Return type

**&#x27;String&#x27;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServerreadyz"></a>
# **apiServerreadyz**
> &#x27;String&#x27; apiServerreadyz()



### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServerreadyz((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
```

### Parameters
This endpoint does not need any parameter.

### Return type

**&#x27;String&#x27;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServervarz"></a>
# **apiServervarz**
> [&#x27;Number&#x27;] apiServervarz()



### Example
```javascript
import {BacalhauClient} from 'bacalhau-client';

let apiInstance = new BacalhauClient.HealthApi();
apiInstance.apiServervarz((error, data, response) => {
  if (error) {
    console.error(error);
  } else {
    console.log('API called successfully. Returned data: ' + data);
  }
});
```

### Parameters
This endpoint does not need any parameter.

### Return type

**[&#x27;Number&#x27;]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

