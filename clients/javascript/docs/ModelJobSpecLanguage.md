# BacalhauClient.ModelJobSpecLanguage

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **String** | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; | [optional] 
**deterministicExecution** | **Boolean** | must this job be run in a deterministic context? | [optional] 
**jobContext** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**language** | **String** | e.g. python | [optional] 
**languageVersion** | **String** | e.g. 3.8 | [optional] 
**programPath** | **String** | optional program path relative to the context dir. one of Command or ProgramPath must be specified | [optional] 
**requirementsPath** | **String** | optional requirements.txt (or equivalent) path relative to the context dir | [optional] 
