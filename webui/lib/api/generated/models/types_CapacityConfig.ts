/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_ResourcesConfig } from './models_ResourcesConfig';
export type types_CapacityConfig = {
    defaultJobResourceLimits?: models_ResourcesConfig;
    ignorePhysicalResourceLimits?: boolean;
    /**
     * Per job amount of resource the system can be using at one time.
     */
    jobResourceLimits?: models_ResourcesConfig;
    /**
     * Total amount of resource the system can be using at one time in aggregate for all jobs.
     */
    totalResourceLimits?: models_ResourcesConfig;
};

