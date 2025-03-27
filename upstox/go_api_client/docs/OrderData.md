# OrderData

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Exchange** | **string** | Exchange to which the order is associated | [optional] [default to null]
**Price** | **float32** | Price at which the order was placed | [optional] [default to null]
**Product** | **string** | Shows if the order was either Intraday, Delivery, CoverOrder or OneCancelsOther | [optional] [default to null]
**Quantity** | **int32** | Quantity with which the order was placed | [optional] [default to null]
**Status** | **string** | Indicates the current status of the order. Valid order status’ are outlined in the table below | [optional] [default to null]
**Tag** | **string** | Tag to uniquely identify an order | [optional] [default to null]
**Validity** | **string** | Order validity (DAY- Day and IOC- Immediate or Cancel (IOC) order) | [optional] [default to null]
**AveragePrice** | **float32** | Average price at which the qty got traded | [optional] [default to null]
**DisclosedQuantity** | **int32** | The quantity that should be disclosed in the market depth | [optional] [default to null]
**ExchangeOrderId** | **string** | Unique order ID assigned by the exchange for the order placed | [optional] [default to null]
**ExchangeTimestamp** | **string** | User readable time at which the order was placed or updated | [optional] [default to null]
**InstrumentToken** | **string** | Identifier issued by Upstox used for subscribing to live market quotes | [optional] [default to null]
**IsAmo** | **bool** | Signifies if the order is an After Market Order | [optional] [default to null]
**StatusMessage** | **string** | Indicates the reason when any order is rejected, not modified or cancelled | [optional] [default to null]
**OrderId** | **string** | Unique order ID assigned internally for the order placed | [optional] [default to null]
**OrderRequestId** | **string** | Apart from 1st order it shows the count of how many requests were sent | [optional] [default to null]
**OrderType** | **string** | Type of order. It can be one of the following MARKET refers to market order&lt;br&gt;LIMIT refers to Limit Order&lt;br&gt;SL refers to Stop Loss Limit&lt;br&gt;SL-M refers to Stop loss market | [optional] [default to null]
**ParentOrderId** | **string** | In case the order is part of the second or third leg of a CO or OCO, the parent order ID is indicated here | [optional] [default to null]
**Tradingsymbol** | **string** | Shows the trading symbol of the instrument | [optional] [default to null]
**TradingSymbol** | **string** | Shows the trading symbol of the instrument | [optional] [default to null]
**OrderTimestamp** | **string** | User readable timestamp at which the order was placed | [optional] [default to null]
**FilledQuantity** | **int32** | The total quantity traded from this particular order | [optional] [default to null]
**TransactionType** | **string** | Indicates whether the order was a buy or sell order | [optional] [default to null]
**TriggerPrice** | **float32** | If the order was a stop loss order then the trigger price set is mentioned here | [optional] [default to null]
**PlacedBy** | **string** | Uniquely identifies the user | [optional] [default to null]
**Variety** | **string** | Order complexity | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

