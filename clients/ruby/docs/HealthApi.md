# SwaggerClient::HealthApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_serverdebug**](HealthApi.md#api_serverdebug) | **GET** /debug | Returns debug information on what the current node is doing.
[**api_serverhealthz**](HealthApi.md#api_serverhealthz) | **GET** /healthz | 
[**api_serverlivez**](HealthApi.md#api_serverlivez) | **GET** /livez | 
[**api_serverlogz**](HealthApi.md#api_serverlogz) | **GET** /logz | 
[**api_serverreadyz**](HealthApi.md#api_serverreadyz) | **GET** /readyz | 
[**api_servervarz**](HealthApi.md#api_servervarz) | **GET** /varz | 

# **api_serverdebug**
> PublicapiDebugResponse api_serverdebug

Returns debug information on what the current node is doing.

### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  #Returns debug information on what the current node is doing.
  result = api_instance.api_serverdebug
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_serverdebug: #{e}"
end
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



# **api_serverhealthz**
> TypesHealthInfo api_serverhealthz



### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  result = api_instance.api_serverhealthz
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_serverhealthz: #{e}"
end
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



# **api_serverlivez**
> String api_serverlivez



### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  result = api_instance.api_serverlivez
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_serverlivez: #{e}"
end
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



# **api_serverlogz**
> String api_serverlogz



### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  result = api_instance.api_serverlogz
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_serverlogz: #{e}"
end
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



# **api_serverreadyz**
> String api_serverreadyz



### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  result = api_instance.api_serverreadyz
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_serverreadyz: #{e}"
end
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



# **api_servervarz**
> Array&lt;Integer&gt; api_servervarz



### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::HealthApi.new

begin
  result = api_instance.api_servervarz
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling HealthApi->api_servervarz: #{e}"
end
```

### Parameters
This endpoint does not need any parameter.

### Return type

**Array&lt;Integer&gt;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json



