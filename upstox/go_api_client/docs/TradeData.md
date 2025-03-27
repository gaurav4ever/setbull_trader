# TradeData

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Exchange** | **string** | Exchange to which the order is associated | [optional] [default to null]
**Product** | **string** | Shows if the order was either Intraday, Delivery, CO or OCO | [optional] [default to null]
**Tradingsymbol** | **string** | Shows the trading symbol which could be a combination of symbol name, instrument, expiry date etc | [optional] [default to null]
**TradingSymbol** | **string** | Shows the trading symbol which could be a combination of symbol name, instrument, expiry date etc | [optional] [default to null]
**InstrumentToken** | **string** | Identifier issued by Upstox used for subscribing to live market quotes | [optional] [default to null]
**OrderType** | **string** | Type of order. It can be one of the following MARKET refers to market order&lt;br&gt;LIMIT refers to Limit Order&lt;br&gt;SL refers to Stop Loss Limit&lt;br&gt;SL-M refers to Stop loss market | [optional] [default to null]
**TransactionType** | **string** | Indicates whether the order was a buy or sell order | [optional] [default to null]
**Quantity** | **int32** | The total quantity traded from this particular order | [optional] [default to null]
**ExchangeOrderId** | **string** | Unique order ID assigned by the exchange for the order placed | [optional] [default to null]
**OrderId** | **string** | Unique order ID assigned internally for the order placed | [optional] [default to null]
**ExchangeTimestamp** | **string** | User readable time at when the trade occurred | [optional] [default to null]
**AveragePrice** | **float32** | Price at which the traded quantity is traded | [optional] [default to null]
**TradeId** | **string** | Trade ID generated from exchange towards traded transaction | [optional] [default to null]
**OrderRefId** | **string** | The order reference ID for which the order must be modified | [optional] [default to null]
**OrderTimestamp** | **string** | User readable timestamp at which the order was placed | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

