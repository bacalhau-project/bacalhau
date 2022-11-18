# HealthApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServer/debug**](HealthApi.md#apiServer/debug) | **GET** /debug | Returns debug information on what the current node is doing.
[**apiServer/healthz**](HealthApi.md#apiServer/healthz) | **GET** /healthz | 
[**apiServer/livez**](HealthApi.md#apiServer/livez) | **GET** /livez | 
[**apiServer/logz**](HealthApi.md#apiServer/logz) | **GET** /logz | 
[**apiServer/readyz**](HealthApi.md#apiServer/readyz) | **GET** /readyz | 
[**apiServer/varz**](HealthApi.md#apiServer/varz) | **GET** /varz | 

<a name="apiServer/debug"></a>
# **apiServer/debug**
> PublicapidebugResponse apiServer/debug()

Returns debug information on what the current node is doing.

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : PublicapidebugResponse = apiInstance.apiServer/debug()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/debug")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/debug")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**PublicapidebugResponse**](PublicapidebugResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

<a name="apiServer/healthz"></a>
# **apiServer/healthz**
> TypesHealthInfo apiServer/healthz()



### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : TypesHealthInfo = apiInstance.apiServer/healthz()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/healthz")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/healthz")
    e.printStackTrace()
}
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

<a name="apiServer/livez"></a>
# **apiServer/livez**
> kotlin.String apiServer/livez()



### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : kotlin.String = apiInstance.apiServer/livez()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/livez")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/livez")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServer/logz"></a>
# **apiServer/logz**
> kotlin.String apiServer/logz()



### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : kotlin.String = apiInstance.apiServer/logz()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/logz")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/logz")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServer/readyz"></a>
# **apiServer/readyz**
> kotlin.String apiServer/readyz()



### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : kotlin.String = apiInstance.apiServer/readyz()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/readyz")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/readyz")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServer/varz"></a>
# **apiServer/varz**
> kotlin.Array&lt;kotlin.Int&gt; apiServer/varz()



### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = HealthApi()
try {
    val result : kotlin.Array<kotlin.Int> = apiInstance.apiServer/varz()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling HealthApi#apiServer/varz")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling HealthApi#apiServer/varz")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.Array&lt;kotlin.Int&gt;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

