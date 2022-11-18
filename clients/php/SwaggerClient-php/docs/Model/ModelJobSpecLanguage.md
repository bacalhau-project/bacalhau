# ModelJobSpecLanguage

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **string** | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; | [optional] 
**deterministic_execution** | **bool** | must this job be run in a deterministic context? | [optional] 
**job_context** | [**\Swagger\Client\Model\ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**language** | **string** | e.g. python | [optional] 
**language_version** | **string** | e.g. 3.8 | [optional] 
**program_path** | **string** | optional program path relative to the context dir. one of Command or ProgramPath must be specified | [optional] 
**requirements_path** | **string** | optional requirements.txt (or equivalent) path relative to the context dir | [optional] 

[[Back to Model list]](../../README.md#documentation-for-models) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to README]](../../README.md)

