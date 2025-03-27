# ModifyOrderRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Quantity** | **int32** | Quantity with which the order was placed | [optional] [default to null]
**Validity** | **string** | Order validity (DAY- Day and IOC- Immediate or Cancel (IOC) order) | [default to null]
**Price** | **float32** | Price at which the order was placed | [default to null]
**OrderId** | **string** | The order ID for which the order must be modified | [default to null]
**OrderType** | **string** | Type of order. It can be one of the following MARKET refers to market order LIMILT refers to Limit Order SL refers to Stop Loss Limit SL-M refers to Stop Loss Market | [default to null]
**DisclosedQuantity** | **int32** | The quantity that should be disclosed in the market depth | [optional] [default to null]
**TriggerPrice** | **float32** | If the order is a stop loss order then the trigger price to be set is mentioned here | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

