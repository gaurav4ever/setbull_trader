# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetFullMarketQuote**](MarketQuoteApi.md#GetFullMarketQuote) | **Get** /v2/market-quote/quotes | Market quotes and instruments - Full market quotes
[**GetMarketQuoteOHLC**](MarketQuoteApi.md#GetMarketQuoteOHLC) | **Get** /v2/market-quote/ohlc | Market quotes and instruments - OHLC quotes
[**Ltp**](MarketQuoteApi.md#Ltp) | **Get** /v2/market-quote/ltp | Market quotes and instruments - LTP quotes.

# **GetFullMarketQuote**
> GetFullMarketQuoteResponse GetFullMarketQuote(ctx, optional)
Market quotes and instruments - Full market quotes

This API provides the functionality to retrieve the full market quotes for one or more instruments.This API returns the complete market data snapshot of up to 500 instruments in one go.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***MarketQuoteApiGetFullMarketQuoteOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MarketQuoteApiGetFullMarketQuoteOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **symbol** | **optional.String**| Comma separated list of symbols | 
 **instrumentKey** | **optional.String**| Comma separated list of instrument keys | 

### Return type

[**GetFullMarketQuoteResponse**](GetFullMarketQuoteResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMarketQuoteOHLC**
> GetMarketQuoteOhlcResponse GetMarketQuoteOHLC(ctx, interval, optional)
Market quotes and instruments - OHLC quotes

This API provides the functionality to retrieve the OHLC quotes for one or more instruments.This API returns the OHLC snapshots of up to 1000 instruments in one go.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **interval** | **string**| Interval to get ohlc data | 
 **optional** | ***MarketQuoteApiGetMarketQuoteOHLCOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MarketQuoteApiGetMarketQuoteOHLCOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **symbol** | **optional.String**| Comma separated list of symbols | 
 **instrumentKey** | **optional.String**| Comma separated list of instrument keys | 

### Return type

[**GetMarketQuoteOhlcResponse**](GetMarketQuoteOHLCResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Ltp**
> GetMarketQuoteLastTradedPriceResponse Ltp(ctx, optional)
Market quotes and instruments - LTP quotes.

This API provides the functionality to retrieve the LTP quotes for one or more instruments.This API returns the LTPs of up to 1000 instruments in one go.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***MarketQuoteApiLtpOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a MarketQuoteApiLtpOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **symbol** | **optional.String**| Comma separated list of symbols | 
 **instrumentKey** | **optional.String**| Comma separated list of instrument keys | 

### Return type

[**GetMarketQuoteLastTradedPriceResponse**](GetMarketQuoteLastTradedPriceResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

