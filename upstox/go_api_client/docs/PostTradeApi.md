# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetTradesByDateRange**](PostTradeApi.md#GetTradesByDateRange) | **Get** /v2/charges/historical-trades | Get historical trades

# **GetTradesByDateRange**
> TradeHistoryResponse GetTradesByDateRange(ctx, startDate, endDate, pageNumber, pageSize, optional)
Get historical trades

This API retrieves the trade history for a specified time interval.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **startDate** | **string**| Date from which trade history needs to be fetched. Date in YYYY-mm-dd format | 
  **endDate** | **string**| Date till which history needs needs to be fetched. Date in YYYY-mm-dd format | 
  **pageNumber** | **int32**| Page number for which you want to fetch trade history  | 
  **pageSize** | **int32**| How many records you want for a page  | 
 **optional** | ***PostTradeApiGetTradesByDateRangeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PostTradeApiGetTradesByDateRangeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **segment** | **optional.String**| Segment for which data is requested can be from the following options EQ - Equity,   FO - Futures and Options,   COM  - Commodity,   CD - Currency Derivatives MF - Mutual Funds | 

### Return type

[**TradeHistoryResponse**](TradeHistoryResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

