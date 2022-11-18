# bacalhau-client.Api.MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiServerid**](MiscApi.md#apiserverid) | **GET** /id | Returns the id of the host node.
[**ApiServerpeers**](MiscApi.md#apiserverpeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**ApiServerversion**](MiscApi.md#apiserverversion) | **POST** /version | Returns the build version running on the server.

<a name="apiserverid"></a>
# **ApiServerid**
> string ApiServerid ()

Returns the id of the host node.

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServeridExample
    {
        public void main()
        {
            var apiInstance = new MiscApi();

            try
            {
                // Returns the id of the host node.
                string result = apiInstance.ApiServerid();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling MiscApi.ApiServerid: " + e.Message );
            }
        }
    }
}
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="apiserverpeers"></a>
# **ApiServerpeers**
> Dictionary<string, List<string>> ApiServerpeers ()

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerpeersExample
    {
        public void main()
        {
            var apiInstance = new MiscApi();

            try
            {
                // Returns the peers connected to the host via the transport layer.
                Dictionary&lt;string, List&lt;string&gt;&gt; result = apiInstance.ApiServerpeers();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling MiscApi.ApiServerpeers: " + e.Message );
            }
        }
    }
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**Dictionary<string, List<string>>**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="apiserverversion"></a>
# **ApiServerversion**
> PublicapiVersionResponse ApiServerversion (PublicapiVersionRequest body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerversionExample
    {
        public void main()
        {
            var apiInstance = new MiscApi();
            var body = new PublicapiVersionRequest(); // PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.

            try
            {
                // Returns the build version running on the server.
                PublicapiVersionResponse result = apiInstance.ApiServerversion(body);
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling MiscApi.ApiServerversion: " + e.Message );
            }
        }
    }
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
