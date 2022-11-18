# ModelJobSpecLanguage

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | [**kotlin.String**](.md) | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; |  [optional]
**deterministicExecution** | [**kotlin.Boolean**](.md) | must this job be run in a deterministic context? |  [optional]
**jobContext** | [**ModelStorageSpec**](ModelStorageSpec.md) |  |  [optional]
**language** | [**kotlin.String**](.md) | e.g. python |  [optional]
**languageVersion** | [**kotlin.String**](.md) | e.g. 3.8 |  [optional]
**programPath** | [**kotlin.String**](.md) | optional program path relative to the context dir. one of Command or ProgramPath must be specified |  [optional]
**requirementsPath** | [**kotlin.String**](.md) | optional requirements.txt (or equivalent) path relative to the context dir |  [optional]
