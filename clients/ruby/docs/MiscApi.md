# SwaggerClient::MiscApi

All URIs are relative to *http://bootstrap.production.bacalhau.org:1234/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_serverid**](MiscApi.md#api_serverid) | **GET** /id | Returns the id of the host node.
[**api_serverpeers**](MiscApi.md#api_serverpeers) | **GET** /peers | Returns the peers connected to the host via the transport layer.
[**api_serverversion**](MiscApi.md#api_serverversion) | **POST** /version | Returns the build version running on the server.

# **api_serverid**
> String api_serverid

Returns the id of the host node.

### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::MiscApi.new

begin
  #Returns the id of the host node.
  result = api_instance.api_serverid
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling MiscApi->api_serverid: #{e}"
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



# **api_serverpeers**
> Hash&lt;String, Array&lt;String&gt;&gt; api_serverpeers

Returns the peers connected to the host via the transport layer.

As described in the [architecture docs](https://docs.bacalhau.org/about-bacalhau/architecture), each node is connected to a number of peer nodes.  Example response: ```json {   \"bacalhau-job-event\": [     \"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL\",     \"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF\",     \"QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG\",     \"QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT\",     \"QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86\"   ] } ```

### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::MiscApi.new

begin
  #Returns the peers connected to the host via the transport layer.
  result = api_instance.api_serverpeers
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling MiscApi->api_serverpeers: #{e}"
end
```

### Parameters
This endpoint does not need any parameter.

### Return type

**Hash&lt;String, Array&lt;String&gt;&gt;**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json



# **api_serverversion**
> PublicapiVersionResponse api_serverversion(body)

Returns the build version running on the server.

See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.

### Example
```ruby
# load the gem
require 'swagger_client'

api_instance = SwaggerClient::MiscApi.new
body = SwaggerClient::PublicapiVersionRequest.new # PublicapiVersionRequest | Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.


begin
  #Returns the build version running on the server.
  result = api_instance.api_serverversion(body)
  p result
rescue SwaggerClient::ApiError => e
  puts "Exception when calling MiscApi->api_serverversion: #{e}"
end
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



