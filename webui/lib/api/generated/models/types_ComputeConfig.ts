/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_JobSelectionPolicy } from './models_JobSelectionPolicy';
import type { types_CapacityConfig } from './types_CapacityConfig';
import type { types_ComputeControlPlaneConfig } from './types_ComputeControlPlaneConfig';
import type { types_DockerCacheConfig } from './types_DockerCacheConfig';
import type { types_JobStoreConfig } from './types_JobStoreConfig';
import type { types_JobTimeoutConfig } from './types_JobTimeoutConfig';
import type { types_LocalPublisherConfig } from './types_LocalPublisherConfig';
import type { types_LoggingConfig } from './types_LoggingConfig';
import type { types_LogStreamConfig } from './types_LogStreamConfig';
export type types_ComputeConfig = {
    capacity?: types_CapacityConfig;
    controlPlaneSettings?: types_ComputeControlPlaneConfig;
    executionStore?: types_JobStoreConfig;
    jobSelection?: models_JobSelectionPolicy;
    jobTimeouts?: types_JobTimeoutConfig;
    localPublisher?: types_LocalPublisherConfig;
    logStreamConfig?: types_LogStreamConfig;
    logging?: types_LoggingConfig;
    manifestCache?: types_DockerCacheConfig;
};

