# LabelSelectorRequirement

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | key is the label key that the selector applies to. | [optional] 
**operator** | [**LabelSelectorRequirementOperator**](LabelSelectorRequirementOperator.md) |  | [optional] 
**values** | **list[str]** | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


