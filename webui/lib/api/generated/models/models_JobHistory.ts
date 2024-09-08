/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_Event } from './models_Event';
import type { models_JobHistoryType } from './models_JobHistoryType';
import type { models_StateChange_models_ExecutionStateType } from './models_StateChange_models_ExecutionStateType';
import type { models_StateChange_models_JobStateType } from './models_StateChange_models_JobStateType';
export type models_JobHistory = {
    Event?: models_Event;
    ExecutionID?: string;
    /**
     * Deprecated: Left for backward compatibility with v1.4.x clients
     */
    ExecutionState?: models_StateChange_models_ExecutionStateType;
    JobID?: string;
    /**
     * TODO: remove with v1.5
     * Deprecated: Left for backward compatibility with v1.4.x clients
     */
    JobState?: models_StateChange_models_JobStateType;
    Time?: string;
    Type?: models_JobHistoryType;
};

