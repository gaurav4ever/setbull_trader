# MultiOrderSummary

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Total** | **int32** | The total number of order lines present in the payload. | [optional] [default to null]
**Success** | **int32** | The number of order lines that were successfully placed without any errors. | [optional] [default to null]
**Error_** | **int32** | The number of order lines that encountered errors during processing, despite their payloads being valid. | [optional] [default to null]
**PayloadError** | **int32** | The number of order lines with payload errors, indicating formatting or data validity issues.&lt;br/&gt;&lt;br/&gt;&lt;b&gt;Note&lt;/b&gt;: Orders are processed only if the entire batch is free of payload_error, ensuring error-free transactions. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

