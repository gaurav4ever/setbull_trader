# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetProfitAndLossCharges**](TradeProfitAndLossApi.md#GetProfitAndLossCharges) | **Get** /v2/trade/profit-loss/charges | Get profit and loss on trades
[**GetTradeWiseProfitAndLossData**](TradeProfitAndLossApi.md#GetTradeWiseProfitAndLossData) | **Get** /v2/trade/profit-loss/data | Get Trade-wise Profit and Loss Report Data
[**GetTradeWiseProfitAndLossMetaData**](TradeProfitAndLossApi.md#GetTradeWiseProfitAndLossMetaData) | **Get** /v2/trade/profit-loss/metadata | Get profit and loss meta data on trades

# **GetProfitAndLossCharges**
> GetProfitAndLossChargesResponse GetProfitAndLossCharges(ctx, segment, financialYear, optional)
Get profit and loss on trades

This API gives the charges incurred by users for their trades

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **segment** | **string**| Segment for which data is requested can be from the following options EQ - Equity,   FO - Futures and Options,   COM  - Commodity,   CD - Currency Derivatives | 
  **financialYear** | **string**| Financial year for which data has been requested. Concatenation of last 2 digits of from year and to year Sample:for 2021-2022, financial_year will be 2122 | 
 **optional** | ***TradeProfitAndLossApiGetProfitAndLossChargesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a TradeProfitAndLossApiGetProfitAndLossChargesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **fromDate** | **optional.String**| Date from which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 
 **toDate** | **optional.String**| Date till which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 

### Return type

[**GetProfitAndLossChargesResponse**](GetProfitAndLossChargesResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTradeWiseProfitAndLossData**
> GetTradeWiseProfitAndLossDataResponse GetTradeWiseProfitAndLossData(ctx, segment, financialYear, pageNumber, pageSize, optional)
Get Trade-wise Profit and Loss Report Data

This API gives the data of the realised Profit and Loss report requested by a user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **segment** | **string**| Segment for which data is requested can be from the following options EQ - Equity,   FO - Futures and Options,   COM  - Commodity,   CD - Currency Derivatives | 
  **financialYear** | **string**| Financial year for which data has been requested. Concatenation of last 2 digits of from year and to year Sample:for 2021-2022, financial_year will be 2122 | 
  **pageNumber** | **int32**| Page Number, the pages are starting from 1 | 
  **pageSize** | **int32**| Page size for pagination. The maximum page size value is obtained from P and L report metadata API. | 
 **optional** | ***TradeProfitAndLossApiGetTradeWiseProfitAndLossDataOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a TradeProfitAndLossApiGetTradeWiseProfitAndLossDataOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **fromDate** | **optional.String**| Date from which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 
 **toDate** | **optional.String**| Date till which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 

### Return type

[**GetTradeWiseProfitAndLossDataResponse**](GetTradeWiseProfitAndLossDataResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTradeWiseProfitAndLossMetaData**
> GetTradeWiseProfitAndLossMetaDataResponse GetTradeWiseProfitAndLossMetaData(ctx, segment, financialYear, optional)
Get profit and loss meta data on trades

This API gives the data of the realised Profit and Loss report requested by a user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **segment** | **string**| Segment for which data is requested can be from the following options EQ - Equity,   FO - Futures and Options,   COM  - Commodity,   CD - Currency Derivatives | 
  **financialYear** | **string**| Financial year for which data has been requested. Concatenation of last 2 digits of from year and to year Sample:for 2021-2022, financial_year will be 2122 | 
 **optional** | ***TradeProfitAndLossApiGetTradeWiseProfitAndLossMetaDataOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a TradeProfitAndLossApiGetTradeWiseProfitAndLossMetaDataOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **fromDate** | **optional.String**| Date from which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 
 **toDate** | **optional.String**| Date till which data needs to be fetched. from_date and to_date should fall under the same financial year as mentioned in financial_year attribute. Date in dd-mm-yyyy format | 

### Return type

[**GetTradeWiseProfitAndLossMetaDataResponse**](GetTradeWiseProfitAndLossMetaDataResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

