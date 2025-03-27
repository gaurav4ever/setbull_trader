# GttModifyOrderRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type_** | **string** | Type of GTT order. It can be one of the following: SINGLE refers to a single-leg GTT order MULTIPLE refers to a multi-leg GTT order | [default to null]
**Quantity** | **int32** | Quantity with which the order is to be placed | [default to null]
**Rules** | [**[]GttRule**](GttRule.md) | List of rules defining the conditions for each leg in the GTT order | [default to null]
**GttOrderId** | **string** | Unique identifier of the GTT order to be modified | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

