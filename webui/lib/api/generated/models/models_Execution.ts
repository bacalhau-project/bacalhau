/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_AllocatedResources } from './models_AllocatedResources';
import type { models_Job } from './models_Job';
import type { models_RunCommandResult } from './models_RunCommandResult';
import type { models_SpecConfig } from './models_SpecConfig';
import type { models_State_models_ExecutionDesiredStateType } from './models_State_models_ExecutionDesiredStateType';
import type { models_State_models_ExecutionStateType } from './models_State_models_ExecutionStateType';
export type models_Execution = {
    /**
     * AllocatedResources is the total resources allocated for the execution tasks.
     */
    AllocatedResources?: models_AllocatedResources;
    /**
     * ComputeState observed state of the execution on the compute node
     */
    ComputeState?: models_State_models_ExecutionStateType;
    /**
     * CreateTime is the time the execution has finished scheduling and been
     * verified by the plan applier.
     */
    CreateTime?: number;
    /**
     * DesiredState of the execution on the compute node
     */
    DesiredState?: models_State_models_ExecutionDesiredStateType;
    /**
     * ID of the evaluation that generated this execution
     */
    EvalID?: string;
    /**
     * FollowupEvalID captures a follow up evaluation created to handle a failed execution
     * that can be rescheduled in the future
     */
    FollowupEvalID?: string;
    /**
     * ID of the execution (UUID)
     */
    ID?: string;
    /**
     * TODO: evaluate using a copy of the job instead of a pointer
     */
    Job?: models_Job;
    /**
     * Job is the parent job of the task being allocated.
     * This is copied at execution time to avoid issues if the job
     * definition is updated.
     */
    JobID?: string;
    /**
     * ModifyTime is the time the execution was last updated.
     */
    ModifyTime?: number;
    /**
     * Name is a logical name of the execution.
     */
    Name?: string;
    /**
     * Namespace is the namespace the execution is created in
     */
    Namespace?: string;
    /**
     * NextExecution is the execution that this execution is being replaced by
     */
    NextExecution?: string;
    /**
     * NodeID is the node this is being placed on
     */
    NodeID?: string;
    /**
     * PreviousExecution is the execution that this execution is replacing
     */
    PreviousExecution?: string;
    /**
     * the published results for this execution
     */
    PublishedResult?: models_SpecConfig;
    /**
     * Revision is increment each time the execution is updated.
     */
    Revision?: number;
    /**
     * RunOutput is the output of the run command
     * TODO: evaluate removing this from execution spec in favour of calling `bacalhau job logs`
     */
    RunOutput?: models_RunCommandResult;
};

