# {{classname}}

All URIs are relative to *https://api-v2.upstox.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CancelGTTOrder**](OrderControllerV3Api.md#CancelGTTOrder) | **Delete** /v3/order/gtt/cancel | Cancel GTT order
[**CancelOrder**](OrderControllerV3Api.md#CancelOrder) | **Delete** /v3/order/cancel | 
[**GetGttOrderDetails**](OrderControllerV3Api.md#GetGttOrderDetails) | **Get** /v3/order/gtt | Get GTT order details
[**ModifyGTTOrder**](OrderControllerV3Api.md#ModifyGTTOrder) | **Put** /v3/order/gtt/modify | Modify GTT order
[**ModifyOrder**](OrderControllerV3Api.md#ModifyOrder) | **Put** /v3/order/modify | 
[**PlaceGTTOrder**](OrderControllerV3Api.md#PlaceGTTOrder) | **Post** /v3/order/gtt/place | Place GTT order
[**PlaceOrder**](OrderControllerV3Api.md#PlaceOrder) | **Post** /v3/order/place | 

# **CancelGTTOrder**
> GttTriggerOrderResponse CancelGTTOrder(ctx, body)
Cancel GTT order

This API allows you to cancel GTT orders.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**GttCancelOrderRequest**](GttCancelOrderRequest.md)|  | 

### Return type

[**GttTriggerOrderResponse**](GttTriggerOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CancelOrder**
> CancelOrderV3Response CancelOrder(ctx, orderId, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orderId** | **string**|  | 
 **optional** | ***OrderControllerV3ApiCancelOrderOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderControllerV3ApiCancelOrderOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **origin** | **optional.String**|  | 

### Return type

[**CancelOrderV3Response**](CancelOrderV3Response.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetGttOrderDetails**
> GetGttOrderResponse GetGttOrderDetails(ctx, optional)
Get GTT order details

GTT_ORDER_DESCRIPTION

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***OrderControllerV3ApiGetGttOrderDetailsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderControllerV3ApiGetGttOrderDetailsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gttOrderId** | **optional.String**| Unique identifier of the GTT order for which the order history is required | 

### Return type

[**GetGttOrderResponse**](GetGttOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ModifyGTTOrder**
> GttTriggerOrderResponse ModifyGTTOrder(ctx, body)
Modify GTT order

This API allows you to modify GTT orders.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**GttModifyOrderRequest**](GttModifyOrderRequest.md)|  | 

### Return type

[**GttTriggerOrderResponse**](GttTriggerOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ModifyOrder**
> ModifyOrderV3Response ModifyOrder(ctx, body, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ModifyOrderRequest**](ModifyOrderRequest.md)|  | 
 **optional** | ***OrderControllerV3ApiModifyOrderOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderControllerV3ApiModifyOrderOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **origin** | **optional.**|  | 

### Return type

[**ModifyOrderV3Response**](ModifyOrderV3Response.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PlaceGTTOrder**
> GttTriggerOrderResponse PlaceGTTOrder(ctx, body)
Place GTT order

This API allows you to place GTT orders.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**GttPlaceOrderRequest**](GttPlaceOrderRequest.md)|  | 

### Return type

[**GttTriggerOrderResponse**](GttTriggerOrderResponse.md)

### Authorization

[OAUTH2](../README.md#OAUTH2)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PlaceOrder**
> PlaceOrderV3Response PlaceOrder(ctx, body, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**PlaceOrderV3Request**](PlaceOrderV3Request.md)|  | 
 **optional** | ***OrderControllerV3ApiPlaceOrderOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a OrderControllerV3ApiPlaceOrderOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **origin** | **optional.**|  | 

### Return type

[**PlaceOrderV3Response**](PlaceOrderV3Response.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

