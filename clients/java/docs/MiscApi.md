# MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServerid**](MiscApi.md#apiServerid) | **GET** /id | Returns the id of the host node.
[**apiServerpeers**](MiscApi.md#apiServerpeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**apiServerversion**](MiscApi.md#apiServerversion) | **POST** /version | Returns the build version running on the server.

<a name="apiServerid"></a>
# **apiServerid**
> String apiServerid()

Returns the id of the host node.

### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.MiscApi;


MiscApi apiInstance = new MiscApi();
try {
    String result = apiInstance.apiServerid();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling MiscApi#apiServerid");
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

<a name="apiServerpeers"></a>
# **apiServerpeers**
> Map&lt;String, List&lt;String&gt;&gt; apiServerpeers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: &#x60;&#x60;&#x60;json {   \&quot;bacalhau-job-event\&quot;: [     \&quot;QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\&quot;,     \&quot;QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\&quot;,     \&quot;QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\&quot;,     \&quot;QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\&quot;,     \&quot;QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\&quot;   ] } &#x60;&#x60;&#x60;

### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.MiscApi;


MiscApi apiInstance = new MiscApi();
try {
    Map<String, List<String>> result = apiInstance.apiServerpeers();
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling MiscApi#apiServerpeers");
    e.printStackTrace();
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**Map&lt;String, List&lt;String&gt;&gt;**](List.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

<a name="apiServerversion"></a>
# **apiServerversion**
> PublicapiVersionResponse apiServerversion(body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of &#x60;gitversion&#x60; tags.

### Example
```java
// Import classes:
//import io.swagger.client.ApiException;
//import io.swagger.client.api.MiscApi;


MiscApi apiInstance = new MiscApi();
PublicapiVersionRequest body = new PublicapiVersionRequest(); // PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
try {
    PublicapiVersionResponse result = apiInstance.apiServerversion(body);
    System.out.println(result);
} catch (ApiException e) {
    System.err.println("Exception when calling MiscApi#apiServerversion");
    e.printStackTrace();
}
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

