# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetProfile**](UserApi.md#GetProfile) | **Get** /v2/user/profile | Get profile
[**GetUserFundMargin**](UserApi.md#GetUserFundMargin) | **Get** /v2/user/get-funds-and-margin | Get User Fund And Margin

# **GetProfile**
> GetProfileResponse GetProfile(ctx, )
Get profile

This API allows to fetch the complete information of the user who is logged in including the products, order types and exchanges enabled for the user

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**GetProfileResponse**](GetProfileResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetUserFundMargin**
> GetUserFundMarginResponse GetUserFundMargin(ctx, optional)
Get User Fund And Margin

Shows the balance of the user in equity and commodity market.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserApiGetUserFundMarginOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserApiGetUserFundMarginOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **segment** | **optional.String**|  | 

### Return type

[**GetUserFundMarginResponse**](GetUserFundMarginResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

