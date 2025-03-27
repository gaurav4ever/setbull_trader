# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CancelMultiOrder**](OrderApi.md#CancelMultiOrder) | **Delete** /v2/order/multi/cancel | Cancel multi order
[**CancelOrder1**](OrderApi.md#CancelOrder1) | **Delete** /v2/order/cancel | Cancel order
[**ExitPositions**](OrderApi.md#ExitPositions) | **Post** /v2/order/positions/exit | Exit all positions
[**GetOrderBook**](OrderApi.md#GetOrderBook) | **Get** /v2/order/retrieve-all | Get order book
[**GetOrderDetails**](OrderApi.md#GetOrderDetails) | **Get** /v2/order/history | Get order history
[**GetOrderStatus**](OrderApi.md#GetOrderStatus) | **Get** /v2/order/details | Get order details
[**GetTradeHistory**](OrderApi.md#GetTradeHistory) | **Get** /v2/order/trades/get-trades-for-day | Get trades
[**GetTradesByOrder**](OrderApi.md#GetTradesByOrder) | **Get** /v2/order/trades | Get trades for order
[**ModifyOrder1**](OrderApi.md#ModifyOrder1) | **Put** /v2/order/modify | Modify order
[**PlaceMultiOrder**](OrderApi.md#PlaceMultiOrder) | **Post** /v2/order/multi/place | Place multi order
[**PlaceOrder1**](OrderApi.md#PlaceOrder1) | **Post** /v2/order/place | Place order

# **CancelMultiOrder**
> CancelOrExitMultiOrderResponse CancelMultiOrder(ctx, optional)
Cancel multi order

API to cancel all the open or pending orders which can be applied to both AMO and regular orders.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***OrderApiCancelMultiOrderOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderApiCancelMultiOrderOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tag** | **optional.String**| The tag associated with the orders for which the orders must be cancelled | 
 **segment** | **optional.String**| The segment for which the orders must be cancelled | 

### Return type

[**CancelOrExitMultiOrderResponse**](CancelOrExitMultiOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CancelOrder1**
> CancelOrderResponse CancelOrder1(ctx, orderId)
Cancel order

Cancel order API can be used for two purposes: Cancelling an open order which could be an AMO or a normal order It is also used to EXIT a CO or OCO(bracket order)

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orderId** | **string**| The order ID for which the order must be cancelled | 

### Return type

[**CancelOrderResponse**](CancelOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ExitPositions**
> CancelOrExitMultiOrderResponse ExitPositions(ctx, optional)
Exit all positions

This API provides the functionality to exit all the positions 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***OrderApiExitPositionsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderApiExitPositionsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tag** | **optional.String**| The tag associated with the positions for which the positions must be exit | 
 **segment** | **optional.String**| The segment for which the positions must be exit | 

### Return type

[**CancelOrExitMultiOrderResponse**](CancelOrExitMultiOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetOrderBook**
> GetOrderBookResponse GetOrderBook(ctx, )
Get order book

This API provides the list of orders placed by the user. The orders placed by the user is transient for a day and is cleared by the end of the trading session. This API returns all states of the orders, namely, open, pending, and filled ones.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**GetOrderBookResponse**](GetOrderBookResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetOrderDetails**
> GetOrderResponse GetOrderDetails(ctx, optional)
Get order history

This API provides the details of the particular order the user has placed. The orders placed by the user is transient for a day and are cleared by the end of the trading session. This API returns all states of the orders, namely, open, pending, and filled ones.  The order history can be requested either using order_id or tag.  If both the options are passed, the history of the order which precisely matches both the order_id and tag will be returned in the response.  If only the tag is passed, the history of all the associated orders which matches the tag will be shared in the response.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***OrderApiGetOrderDetailsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderApiGetOrderDetailsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **orderId** | **optional.String**| The order reference ID for which the order history is required | 
 **tag** | **optional.String**| The unique tag of the order for which the order history is being requested | 

### Return type

[**GetOrderResponse**](GetOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetOrderStatus**
> GetOrderDetailsResponse GetOrderStatus(ctx, optional)
Get order details

This API provides the recent detail of the particular order the user has placed. The orders placed by the user is transient for a day and are cleared by the end of the trading session.\\n\\nThe order details can be requested using order_id.  

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***OrderApiGetOrderStatusOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderApiGetOrderStatusOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **orderId** | **optional.String**| The order reference ID for which the order details is required | 

### Return type

[**GetOrderDetailsResponse**](GetOrderDetailsResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTradeHistory**
> GetTradeResponse GetTradeHistory(ctx, )
Get trades

Retrieve the trades executed for the day

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**GetTradeResponse**](GetTradeResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTradesByOrder**
> GetTradeResponse GetTradesByOrder(ctx, orderId)
Get trades for order

Retrieve the trades executed for an order

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orderId** | **string**| The order ID for which the order to get order trades | 

### Return type

[**GetTradeResponse**](GetTradeResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ModifyOrder1**
> ModifyOrderResponse ModifyOrder1(ctx, body)
Modify order

This API allows you to modify an order. For modification orderId is mandatory. With orderId you need to send the optional parameter which needs to be modified. In case the optional parameters aren't sent, the default will be considered from the original order

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ModifyOrderRequest**](ModifyOrderRequest.md)|  | 

### Return type

[**ModifyOrderResponse**](ModifyOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PlaceMultiOrder**
> MultiOrderResponse PlaceMultiOrder(ctx, body)
Place multi order

This API allows you to place multiple orders to the exchange via Upstox.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**[]MultiOrderRequest**](MultiOrderRequest.md)|  | 

### Return type

[**MultiOrderResponse**](MultiOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PlaceOrder1**
> PlaceOrderResponse PlaceOrder1(ctx, body)
Place order

This API allows you to place an order to the exchange via Upstox.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**PlaceOrderRequest**](PlaceOrderRequest.md)|  | 

### Return type

[**PlaceOrderResponse**](PlaceOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

