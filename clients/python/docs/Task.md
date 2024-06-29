# Task

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**engine** | [**SpecConfig**](SpecConfig.md) |  | [optional] 
**env** | **dict(str, str)** | Map of environment variables to be used by the driver | [optional] 
**input_sources** | [**list[InputSource]**](InputSource.md) | InputSources is a list of remote artifacts to be downloaded before running the task and mounted into the task. | [optional] 
**meta** | **dict(str, str)** | Meta is used to associate arbitrary metadata with this task. | [optional] 
**name** | **str** | Name of the task | [optional] 
**network** | [**NetworkConfig**](NetworkConfig.md) |  | [optional] 
**publisher** | [**SpecConfig**](SpecConfig.md) |  | [optional] 
**resources** | **AllOfTaskResources** | ResourcesConfig is the resources needed by this task | [optional] 
**result_paths** | [**list[ResultPath]**](ResultPath.md) | ResultPaths is a list of task volumes to be included in the task&#x27;s published result | [optional] 
**timeouts** | [**TimeoutConfig**](TimeoutConfig.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

