# MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServer/id**](MiscApi.md#apiServer/id) | **GET** /id | Returns the id of the host node.
[**apiServer/peers**](MiscApi.md#apiServer/peers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**apiServer/version**](MiscApi.md#apiServer/version) | **POST** /version | Returns the build version running on the server.

<a name="apiServer/id"></a>
# **apiServer/id**
> kotlin.String apiServer/id()

Returns the id of the host node.

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = MiscApi()
try {
    val result : kotlin.String = apiInstance.apiServer/id()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling MiscApi#apiServer/id")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling MiscApi#apiServer/id")
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

<a name="apiServer/peers"></a>
# **apiServer/peers**
> kotlin.collections.Map&lt;kotlin.String, kotlin.Array&lt;kotlin.String&gt;&gt; apiServer/peers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: &#x60;&#x60;&#x60;json {   \&quot;bacalhau-job-event\&quot;: [     \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,     \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,     \&quot;QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\&quot;,     \&quot;QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\&quot;,     \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;   ] } &#x60;&#x60;&#x60;

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = MiscApi()
try {
    val result : kotlin.collections.Map<kotlin.String, kotlin.Array<kotlin.String>> = apiInstance.apiServer/peers()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling MiscApi#apiServer/peers")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling MiscApi#apiServer/peers")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.collections.Map&lt;kotlin.String, kotlin.Array&lt;kotlin.String&gt;&gt;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

<a name="apiServer/version"></a>
# **apiServer/version**
> PublicapiversionResponse apiServer/version(body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of &#x60;gitversion&#x60; tags.

### Example
```kotlin
// Import classes:
//import bacalhau-client.infrastructure.*
//import io.swagger.client.models.*;

val apiInstance = MiscApi()
val body : PublicapiversionRequest =  // PublicapiversionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
try {
    val result : PublicapiversionResponse = apiInstance.apiServer/version(body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling MiscApi#apiServer/version")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling MiscApi#apiServer/version")
    e.printStackTrace()
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**PublicapiversionRequest**](PublicapiversionRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. |

### Return type

[**PublicapiversionResponse**](PublicapiversionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

