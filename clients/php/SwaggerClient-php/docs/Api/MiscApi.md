# Swagger\Client\MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServerid**](MiscApi.md#apiserverid) | **GET** /id | Returns the id of the host node.
[**apiServerpeers**](MiscApi.md#apiserverpeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**apiServerversion**](MiscApi.md#apiserverversion) | **POST** /version | Returns the build version running on the server.

# **apiServerid**
> string apiServerid()

Returns the id of the host node.

### Example
```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');

$apiInstance = new Swagger\Client\Api\MiscApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client()
);

try {
    $result = $apiInstance->apiServerid();
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling MiscApi->apiServerid: ', $e->getMessage(), PHP_EOL;
}
?>
```

### Parameters
This endpoint does not need any parameter.

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

[[Back to top]](#) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to Model list]](../../README.md#documentation-for-models) [[Back to README]](../../README.md)

# **apiServerpeers**
> map[string,string[]] apiServerpeers()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');

$apiInstance = new Swagger\Client\Api\MiscApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client()
);

try {
    $result = $apiInstance->apiServerpeers();
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling MiscApi->apiServerpeers: ', $e->getMessage(), PHP_EOL;
}
?>
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**map[string,string[]]**](../Model/array.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to Model list]](../../README.md#documentation-for-models) [[Back to README]](../../README.md)

# **apiServerversion**
> \Swagger\Client\Model\PublicapiVersionResponse apiServerversion($body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');

$apiInstance = new Swagger\Client\Api\MiscApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client()
);
$body = new \Swagger\Client\Model\PublicapiVersionRequest(); // \Swagger\Client\Model\PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

try {
    $result = $apiInstance->apiServerversion($body);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling MiscApi->apiServerversion: ', $e->getMessage(), PHP_EOL;
}
?>
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**\Swagger\Client\Model\PublicapiVersionRequest**](../Model/PublicapiVersionRequest.md)| Request must specify a &#x60;client_id&#x60;. To retrieve your &#x60;client_id&#x60;, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run &#x60;bacalhau describe &lt;job-id&gt;&#x60; and fetch the &#x60;ClientID&#x60; field. |

### Return type

[**\Swagger\Client\Model\PublicapiVersionResponse**](../Model/PublicapiVersionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to Model list]](../../README.md#documentation-for-models) [[Back to README]](../../README.md)

