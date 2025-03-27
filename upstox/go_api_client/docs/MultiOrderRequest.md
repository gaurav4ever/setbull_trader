# MultiOrderRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Quantity** | **int32** | Quantity with which the order is to be placed | [default to null]
**Product** | **string** | Signifies if the order was either Intraday, Delivery, CO or OCO | [default to null]
**Validity** | **string** | It can be one of the following - DAY(default), IOC | [default to null]
**Price** | **float32** | Price at which the order will be placed | [default to null]
**Tag** | **string** |  | [optional] [default to null]
**Slice** | **bool** | To divide the order line based on predefined exchange definitions | [default to null]
**InstrumentToken** | **string** | Key of the instrument | [default to null]
**OrderType** | **string** | Type of order. It can be one of the following MARKET refers to market order LIMIT refers to Limit Order SL refers to Stop Loss Limit SL-M refers to Stop Loss Market | [default to null]
**TransactionType** | **string** | Indicates whether its a buy or sell order | [default to null]
**DisclosedQuantity** | **int32** | The quantity that should be disclosed in the market depth | [default to null]
**TriggerPrice** | **float32** | If the order is a stop loss order then the trigger price to be set is mentioned here | [default to null]
**IsAmo** | **bool** | Signifies if the order is an After Market Order | [default to null]
**CorrelationId** | **string** | A unique identifier for tracking individual orders within the batch | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

