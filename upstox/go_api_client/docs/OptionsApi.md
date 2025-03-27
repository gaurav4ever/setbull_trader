# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetOptionContracts**](OptionsApi.md#GetOptionContracts) | **Get** /v2/option/contract | Get option contracts
[**GetPutCallOptionChain**](OptionsApi.md#GetPutCallOptionChain) | **Get** /v2/option/chain | Get option chain

# **GetOptionContracts**
> GetOptionContractResponse GetOptionContracts(ctx, instrumentKey, optional)
Get option contracts

This API provides the functionality to retrieve the option contracts

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentKey** | **string**| Instrument key for an underlying symbol | 
 **optional** | ***OptionsApiGetOptionContractsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OptionsApiGetOptionContractsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **expiryDate** | **optional.String**| Expiry date in format: YYYY-mm-dd | 

### Return type

[**GetOptionContractResponse**](GetOptionContractResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPutCallOptionChain**
> GetOptionChainResponse GetPutCallOptionChain(ctx, instrumentKey, expiryDate)
Get option chain

This API provides the functionality to retrieve the option chain

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentKey** | **string**| Instrument key for an underlying symbol | 
  **expiryDate** | **string**| Expiry date in format: YYYY-mm-dd | 

### Return type

[**GetOptionChainResponse**](GetOptionChainResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

