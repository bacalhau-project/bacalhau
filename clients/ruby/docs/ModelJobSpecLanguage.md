# SwaggerClient::ModelJobSpecLanguage

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **String** | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; | [optional] 
**deterministic_execution** | **BOOLEAN** | must this job be run in a deterministic context? | [optional] 
**job_context** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**language** | **String** | e.g. python | [optional] 
**language_version** | **String** | e.g. 3.8 | [optional] 
**program_path** | **String** | optional program path relative to the context dir. one of Command or ProgramPath must be specified | [optional] 
**requirements_path** | **String** | optional requirements.txt (or equivalent) path relative to the context dir | [optional] 

