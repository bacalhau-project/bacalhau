# HealthApi

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
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    PublicapiDebugResponse result = apiInstance.apiServerdebug();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServerdebug");
    e.printStackTrace();
}
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
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    TypesHealthInfo result = apiInstance.apiServerhealthz();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServerhealthz");
    e.printStackTrace();
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

<a name="apiServerlivez"></a>
# **apiServerlivez**
> String apiServerlivez()



### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    String result = apiInstance.apiServerlivez();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServerlivez");
    e.printStackTrace();
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServerlogz"></a>
# **apiServerlogz**
> String apiServerlogz()



### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    String result = apiInstance.apiServerlogz();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServerlogz");
    e.printStackTrace();
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServerreadyz"></a>
# **apiServerreadyz**
> String apiServerreadyz()



### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    String result = apiInstance.apiServerreadyz();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServerreadyz");
    e.printStackTrace();
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**String**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

<a name="apiServervarz"></a>
# **apiServervarz**
> List&lt;Integer&gt; apiServervarz()



### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.HealthApi;


HealthApi apiInstance = new HealthApi();
try {
    List<Integer> result = apiInstance.apiServervarz();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling HealthApi#apiServervarz");
    e.printStackTrace();
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**List&lt;Integer&gt;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

