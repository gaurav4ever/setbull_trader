# TokenResponse

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Email** | **string** | E-mail address of the user | [optional] [default to null]
**Exchanges** | **[]string** | Lists the exchanges to which the user has access | [optional] [default to null]
**Products** | **[]string** | Lists the products types to which the user has access | [optional] [default to null]
**Broker** | **string** | The broker ID | [optional] [default to null]
**UserId** | **string** | Uniquely identifies the user | [optional] [default to null]
**UserName** | **string** | Name of the user | [optional] [default to null]
**OrderTypes** | **[]string** | Order types enabled for the user | [optional] [default to null]
**UserType** | **string** |   Identifies the user&#x27;s registered role at the broker. This will be individual for all retail users | [optional] [default to null]
**Poa** | **bool** |   To depict if the user has given power of attorney for transactions | [optional] [default to null]
**Ddpi** | **bool** |   Indicates if DDPI is enabled for trading | [optional] [default to null]
**IsActive** | **bool** |   Whether the status of account is active or not | [optional] [default to null]
**AccessToken** | **string** | The authentication token that is to used with every subsequent API requests | [optional] [default to null]
**ExtendedToken** | **string** | An extended authentication token with a prolonged validity period, intended for specific API requests. Ensure you use this token only with the designated set of APIs. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

