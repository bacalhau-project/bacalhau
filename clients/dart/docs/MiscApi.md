# swagger.api.MiscApi

## Load the API package
```dart
import 'package:swagger/api.dart';
```

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServerId**](MiscApi.md#apiServerId) | **GET** /id | Returns the id of the host node.
[**apiServerPeers**](MiscApi.md#apiServerPeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**apiServerVersion**](MiscApi.md#apiServerVersion) | **POST** /version | Returns the build version running on the server.

# **apiServerId**
> String apiServerId()

Returns the id of the host node.

### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new MiscApi();

try {
    var result = api_instance.apiServerId();
    print(result);
} catch (e) {
    print("Exception when calling MiscApi->apiServerId: $e\n");
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **apiServerPeers**
> Map<String, List<String>> apiServerPeers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new MiscApi();

try {
    var result = api_instance.apiServerPeers();
    print(result);
} catch (e) {
    print("Exception when calling MiscApi->apiServerPeers: $e\n");
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**Map<String, List<String>>**](List.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **apiServerVersion**
> PublicapiVersionResponse apiServerVersion(body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new MiscApi();
var body = new PublicapiVersionRequest(); // PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try {
    var result = api_instance.apiServerVersion(body);
    print(result);
} catch (e) {
    print("Exception when calling MiscApi->apiServerVersion: $e\n");
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

