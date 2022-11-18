# bacalhau-client.Model.ModelJobSpecLanguage
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Command** | **string** | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; | [optional] 
**DeterministicExecution** | **bool?** | must this job be run in a deterministic context? | [optional] 
**JobContext** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**Language** | **string** | e.g. python | [optional] 
**LanguageVersion** | **string** | e.g. 3.8 | [optional] 
**ProgramPath** | **string** | optional program path relative to the context dir. one of Command or ProgramPath must be specified | [optional] 
**RequirementsPath** | **string** | optional requirements.txt (or equivalent) path relative to the context dir | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

