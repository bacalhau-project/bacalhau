/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_Resources } from './models_Resources';
export type models_ComputeNodeInfo = {
    AvailableCapacity?: models_Resources;
    EnqueuedExecutions?: number;
    ExecutionEngines?: Array<string>;
    MaxCapacity?: models_Resources;
    MaxJobRequirements?: models_Resources;
    Publishers?: Array<string>;
    QueueCapacity?: models_Resources;
    RunningExecutions?: number;
    StorageSources?: Array<string>;
};

