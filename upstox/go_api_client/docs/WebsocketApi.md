# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetMarketDataFeed**](WebsocketApi.md#GetMarketDataFeed) | **Get** /v2/feed/market-data-feed | Market Data Feed
[**GetMarketDataFeedAuthorize**](WebsocketApi.md#GetMarketDataFeedAuthorize) | **Get** /v2/feed/market-data-feed/authorize | Market Data Feed Authorize
[**GetPortfolioStreamFeed**](WebsocketApi.md#GetPortfolioStreamFeed) | **Get** /v2/feed/portfolio-stream-feed | Portfolio Stream Feed
[**GetPortfolioStreamFeedAuthorize**](WebsocketApi.md#GetPortfolioStreamFeedAuthorize) | **Get** /v2/feed/portfolio-stream-feed/authorize | Portfolio Stream Feed Authorize

# **GetMarketDataFeed**
> GetMarketDataFeed(ctx, )
Market Data Feed

 This API redirects the client to the respective socket endpoint to receive Market updates.

### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetMarketDataFeedAuthorize**
> WebsocketAuthRedirectResponse GetMarketDataFeedAuthorize(ctx, )
Market Data Feed Authorize

This API provides the functionality to retrieve the socket endpoint URI for Market updates.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**WebsocketAuthRedirectResponse**](WebsocketAuthRedirectResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPortfolioStreamFeed**
> GetPortfolioStreamFeed(ctx, optional)
Portfolio Stream Feed

This API redirects the client to the respective socket endpoint to receive Portfolio updates.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***WebsocketApiGetPortfolioStreamFeedOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a WebsocketApiGetPortfolioStreamFeedOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **updateTypes** | **optional.String**| Identifiers separated by commas denote the types of updates to receive. | 

### Return type

 (empty response body)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetPortfolioStreamFeedAuthorize**
> WebsocketAuthRedirectResponse GetPortfolioStreamFeedAuthorize(ctx, optional)
Portfolio Stream Feed Authorize

 This API provides the functionality to retrieve the socket endpoint URI for Portfolio updates.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***WebsocketApiGetPortfolioStreamFeedAuthorizeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a WebsocketApiGetPortfolioStreamFeedAuthorizeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **updateTypes** | **optional.String**| Identifiers separated by commas denote the types of updates to receive. | 

### Return type

[**WebsocketAuthRedirectResponse**](WebsocketAuthRedirectResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

