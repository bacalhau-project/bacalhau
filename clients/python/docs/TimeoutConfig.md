# TimeoutConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**execution_timeout** | **int** | ExecutionTimeout is the maximum amount of time a task is allowed to run in seconds. Zero means no timeout, such as for a daemon task. | [optional] 
**queue_timeout** | **int** | QueueTimeout is the maximum amount of time a task is allowed to wait in the orchestrator queue in seconds before being scheduled. Zero means no timeout. | [optional] 
**total_timeout** | **int** | TotalTimeout is the maximum amount of time a task is allowed to complete in seconds. This includes the time spent in the queue, the time spent executing and the time spent retrying. Zero means no timeout. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

