# swagger.api.HealthApi

## Load the API package
```dart
import 'package:swagger/api.dart';
```

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**apiServerDebug**](HealthApi.md#apiServerDebug) | **GET** /debug | Returns debug information on what the current node is doing.
[**apiServerHealthz**](HealthApi.md#apiServerHealthz) | **GET** /healthz | 
[**apiServerLivez**](HealthApi.md#apiServerLivez) | **GET** /livez | 
[**apiServerLogz**](HealthApi.md#apiServerLogz) | **GET** /logz | 
[**apiServerReadyz**](HealthApi.md#apiServerReadyz) | **GET** /readyz | 
[**apiServerVarz**](HealthApi.md#apiServerVarz) | **GET** /varz | 

# **apiServerDebug**
> PublicapiDebugResponse apiServerDebug()

Returns debug information on what the current node is doing.

### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerDebug();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerDebug: $e\n");
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

# **apiServerHealthz**
> TypesHealthInfo apiServerHealthz()



### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerHealthz();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerHealthz: $e\n");
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

# **apiServerLivez**
> String apiServerLivez()



### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerLivez();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerLivez: $e\n");
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

# **apiServerLogz**
> String apiServerLogz()



### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerLogz();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerLogz: $e\n");
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

# **apiServerReadyz**
> String apiServerReadyz()



### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerReadyz();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerReadyz: $e\n");
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

# **apiServerVarz**
> List<int> apiServerVarz()



### Example
```dart
import 'package:swagger/api.dart';

var api_instance = new HealthApi();

try {
    var result = api_instance.apiServerVarz();
    print(result);
} catch (e) {
    print("Exception when calling HealthApi->apiServerVarz: $e\n");
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

**List<int>**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

