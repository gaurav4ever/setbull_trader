# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetExchangeTimings**](MarketHolidaysAndTimingsApi.md#GetExchangeTimings) | **Get** /v2/market/timings/{date} | Get Exchange Timings on particular date
[**GetHoliday**](MarketHolidaysAndTimingsApi.md#GetHoliday) | **Get** /v2/market/holidays/{date} | Get Holiday on particular date
[**GetHolidays**](MarketHolidaysAndTimingsApi.md#GetHolidays) | **Get** /v2/market/holidays | Get Holiday list of current year
[**GetMarketStatus**](MarketHolidaysAndTimingsApi.md#GetMarketStatus) | **Get** /v2/market/status/{exchange} | Get Market status for particular exchange

# **GetExchangeTimings**
> GetExchangeTimingResponse GetExchangeTimings(ctx, date)
Get Exchange Timings on particular date

This API provides the functionality to retrieve the exchange timings on particular date

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **date** | **string**|  | 

### Return type

[**GetExchangeTimingResponse**](GetExchangeTimingResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetHoliday**
> GetHolidayResponse GetHoliday(ctx, date)
Get Holiday on particular date

This API provides the functionality to retrieve the holiday on particular date

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **date** | **string**|  | 

### Return type

[**GetHolidayResponse**](GetHolidayResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetHolidays**
> GetHolidayResponse GetHolidays(ctx, )
Get Holiday list of current year

This API provides the functionality to retrieve the holiday list of current year

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**GetHolidayResponse**](GetHolidayResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMarketStatus**
> GetMarketStatusResponse GetMarketStatus(ctx, exchange)
Get Market status for particular exchange

This API provides the functionality to retrieve the market status for particular exchange

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **exchange** | **string**|  | 

### Return type

[**GetMarketStatusResponse**](GetMarketStatusResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

