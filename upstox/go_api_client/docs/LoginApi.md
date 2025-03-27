# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**Authorize**](LoginApi.md#Authorize) | **Get** /v2/login/authorization/dialog | Authorize API
[**InitTokenRequestForIndieUser**](LoginApi.md#InitTokenRequestForIndieUser) | **Post** /v3/login/auth/token/request/{client_id} | Init token API
[**Logout**](LoginApi.md#Logout) | **Delete** /v2/logout | Logout
[**Token**](LoginApi.md#Token) | **Post** /v2/login/authorization/token | Get token API

# **Authorize**
> Authorize(ctx, clientId, redirectUri, optional)
Authorize API

This provides details on the login endpoint.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **clientId** | **string**|  | 
  **redirectUri** | **string**|  | 
 **optional** | ***LoginApiAuthorizeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a LoginApiAuthorizeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **state** | **optional.String**|  | 
 **scope** | **optional.String**|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InitTokenRequestForIndieUser**
> TokenResponse InitTokenRequestForIndieUser(ctx, body, clientId)
Init token API

This API provides the initialize the generate token and it's expiry for an indie user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**IndieUserTokenRequest**](IndieUserTokenRequest.md)|  | 
  **clientId** | **string**|  | 

### Return type

[**TokenResponse**](TokenResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Logout**
> LogoutResponse Logout(ctx, )
Logout

Logout

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**LogoutResponse**](LogoutResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Token**
> TokenResponse Token(ctx, optional)
Get token API

This API provides the functionality to obtain opaque token from authorization_code exchange and also provides the user’s profile in the same response.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***LoginApiTokenOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a LoginApiTokenOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **code** | **optional.**|  | 
 **clientId** | **optional.**|  | 
 **clientSecret** | **optional.**|  | 
 **redirectUri** | **optional.**|  | 
 **grantType** | **optional.**|  | 

### Return type

[**TokenResponse**](TokenResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/x-www-form-urlencoded
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

