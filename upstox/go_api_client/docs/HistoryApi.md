# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetHistoricalCandleData**](HistoryApi.md#GetHistoricalCandleData) | **Get** /v2/historical-candle/{instrumentKey}/{interval}/{to_date} | Historical candle data
[**GetHistoricalCandleData1**](HistoryApi.md#GetHistoricalCandleData1) | **Get** /v2/historical-candle/{instrumentKey}/{interval}/{to_date}/{from_date} | Historical candle data
[**GetIntraDayCandleData**](HistoryApi.md#GetIntraDayCandleData) | **Get** /v2/historical-candle/intraday/{instrumentKey}/{interval} | Intra day candle data

# **GetHistoricalCandleData**
> GetHistoricalCandleResponse GetHistoricalCandleData(ctx, instrumentKey, interval, toDate)
Historical candle data

Get OHLC values for all instruments across various timeframes. Historical data can be fetched for the following durations. 1minute: last 1 month candles till endDate 30minute: last 1 year candles till endDate day: last 1 year candles till endDate week: last 10 year candles till endDate month: last 10 year candles till endDate

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentKey** | **string**|  | 
  **interval** | **string**|  | 
  **toDate** | **string**|  | 

### Return type

[**GetHistoricalCandleResponse**](GetHistoricalCandleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetHistoricalCandleData1**
> GetHistoricalCandleResponse GetHistoricalCandleData1(ctx, instrumentKey, interval, toDate, fromDate)
Historical candle data

Get OHLC values for all instruments across various timeframes. Historical data can be fetched for the following durations. 1minute: last 1 month candles till endDate 30minute: last 1 year candles till endDate day: last 1 year candles till endDate week: last 10 year candles till endDate month: last 10 year candles till endDate

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentKey** | **string**|  | 
  **interval** | **string**|  | 
  **toDate** | **string**|  | 
  **fromDate** | **string**|  | 

### Return type

[**GetHistoricalCandleResponse**](GetHistoricalCandleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetIntraDayCandleData**
> GetIntraDayCandleResponse GetIntraDayCandleData(ctx, instrumentKey, interval)
Intra day candle data

Get OHLC values for all instruments for the present trading day

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **instrumentKey** | **string**|  | 
  **interval** | **string**|  | 

### Return type

[**GetIntraDayCandleResponse**](GetIntraDayCandleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

