/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_InputSource } from './models_InputSource';
import type { models_NetworkConfig } from './models_NetworkConfig';
import type { models_ResourcesConfig } from './models_ResourcesConfig';
import type { models_ResultPath } from './models_ResultPath';
import type { models_SpecConfig } from './models_SpecConfig';
import type { models_TimeoutConfig } from './models_TimeoutConfig';
export type models_Task = {
    Engine?: models_SpecConfig;
    /**
     * Map of environment variables to be used by the driver
     */
    Env?: Record<string, string>;
    /**
     * InputSources is a list of remote artifacts to be downloaded before running the task
     * and mounted into the task.
     */
    InputSources?: Array<models_InputSource>;
    /**
     * Meta is used to associate arbitrary metadata with this task.
     */
    Meta?: Record<string, string>;
    /**
     * Name of the task
     */
    Name?: string;
    Network?: models_NetworkConfig;
    Publisher?: models_SpecConfig;
    /**
     * ResourcesConfig is the resources needed by this task
     */
    Resources?: models_ResourcesConfig;
    /**
     * ResultPaths is a list of task volumes to be included in the task's published result
     */
    ResultPaths?: Array<models_ResultPath>;
    Timeouts?: models_TimeoutConfig;
};

