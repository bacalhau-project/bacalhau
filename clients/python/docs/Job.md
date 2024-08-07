# Job

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**constraints** | [**list[LabelSelectorRequirement]**](LabelSelectorRequirement.md) | Constraints is a selector which must be true for the compute node to run this job. | [optional] 
**count** | **int** | Count is the number of replicas that should be scheduled. | [optional] 
**create_time** | **int** |  | [optional] 
**id** | **str** | ID is a unique identifier assigned to this job. It helps to distinguish jobs with the same name after they have been deleted and re-created. The ID is generated by the server and should not be set directly by the client. | [optional] 
**labels** | **dict(str, str)** | Labels is used to associate arbitrary labels with this job, which can be used for filtering. key&#x3D;value | [optional] 
**meta** | **dict(str, str)** | Meta is used to associate arbitrary metadata with this job. | [optional] 
**modify_time** | **int** |  | [optional] 
**name** | **str** | Name is the logical name of the job used to refer to it. Submitting a job with the same name as an existing job will result in an update to the existing job. | [optional] 
**namespace** | **str** | Namespace is the namespace this job is running in. | [optional] 
**priority** | **int** | Priority defines the scheduling priority of this job. | [optional] 
**revision** | **int** | Revision is a per-job monotonically increasing revision number that is incremented on each update to the job&#x27;s state or specification | [optional] 
**state** | **AllOfJobState** | State is the current state of the job. | [optional] 
**tasks** | [**list[Task]**](Task.md) |  | [optional] 
**type** | **str** | Type is the type of job this is, e.g. \&quot;daemon\&quot; or \&quot;batch\&quot;. | [optional] 
**version** | **int** | Version is a per-job monotonically increasing version number that is incremented on each job specification update. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

