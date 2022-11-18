# bacalhau-client.Api.HealthApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiServerdebug**](HealthApi.md#apiserverdebug) | **GET** /debug | Returns debug information on what the current node is doing.
[**ApiServerhealthz**](HealthApi.md#apiserverhealthz) | **GET** /healthz | 
[**ApiServerlivez**](HealthApi.md#apiserverlivez) | **GET** /livez | 
[**ApiServerlogz**](HealthApi.md#apiserverlogz) | **GET** /logz | 
[**ApiServerreadyz**](HealthApi.md#apiserverreadyz) | **GET** /readyz | 
[**ApiServervarz**](HealthApi.md#apiservervarz) | **GET** /varz | 

<a name="apiserverdebug"></a>
# **ApiServerdebug**
> PublicapiDebugResponse ApiServerdebug ()

Returns debug information on what the current node is doing.

### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerdebugExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                // Returns debug information on what the current node is doing.
                PublicapiDebugResponse result = apiInstance.ApiServerdebug();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServerdebug: " + e.Message );
            }
        }
    }
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="apiserverhealthz"></a>
# **ApiServerhealthz**
> TypesHealthInfo ApiServerhealthz ()



### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerhealthzExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                TypesHealthInfo result = apiInstance.ApiServerhealthz();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServerhealthz: " + e.Message );
            }
        }
    }
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
<a name="apiserverlivez"></a>
# **ApiServerlivez**
> string ApiServerlivez ()



### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerlivezExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                string result = apiInstance.ApiServerlivez();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServerlivez: " + e.Message );
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
<a name="apiserverlogz"></a>
# **ApiServerlogz**
> string ApiServerlogz ()



### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerlogzExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                string result = apiInstance.ApiServerlogz();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServerlogz: " + e.Message );
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
<a name="apiserverreadyz"></a>
# **ApiServerreadyz**
> string ApiServerreadyz ()



### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServerreadyzExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                string result = apiInstance.ApiServerreadyz();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServerreadyz: " + e.Message );
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
<a name="apiservervarz"></a>
# **ApiServervarz**
> List<int?> ApiServervarz ()



### Example
```csharp
using System;
using System.Diagnostics;
using bacalhau-client.Api;
using bacalhau-client.Client;
using bacalhau-client.Model;

namespace Example
{
    public class ApiServervarzExample
    {
        public void main()
        {
            var apiInstance = new HealthApi();

            try
            {
                List&lt;int?&gt; result = apiInstance.ApiServervarz();
                Debug.WriteLine(result);
            }
            catch (Exception e)
            {
                Debug.Print("Exception when calling HealthApi.ApiServervarz: " + e.Message );
            }
        }
    }
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**List<int?>**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
