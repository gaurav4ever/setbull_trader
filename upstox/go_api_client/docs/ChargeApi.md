# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetBrokerage**](ChargeApi.md#GetBrokerage) | **Get** /v2/charges/brokerage | Brokerage details
[**PostMargin**](ChargeApi.md#PostMargin) | **Post** /v2/charges/margin | Calculate Margin

# **GetBrokerage**
> GetBrokerageResponse GetBrokerage(ctx, instrumentToken, quantity, product, transactionType, price)
Brokerage details

Compute Brokerage Charges

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentToken** | **string**| Key of the instrument | 
  **quantity** | **int32**| Quantity with which the order is to be placed | 
  **product** | **string**| Product with which the order is to be placed | 
  **transactionType** | **string**| Indicates whether its a BUY or SELL order | 
  **price** | **float32**| Price with which the order is to be placed | 

### Return type

[**GetBrokerageResponse**](GetBrokerageResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostMargin**
> PostMarginResponse PostMargin(ctx, body)
Calculate Margin

Compute Margin

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**MarginRequest**](MarginRequest.md)|  | 

### Return type

[**PostMarginResponse**](PostMarginResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

