/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration } from './github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration';
export type types_JobTimeoutConfig = {
    /**
     * DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
     * no timeout requirement defined.
     */
    defaultJobExecutionTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
     * check.
     */
    jobExecutionTimeoutClientIDBypassList?: Array<string>;
    /**
     * JobNegotiationTimeout default timeout value to hold a bid for a job
     */
    jobNegotiationTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
     * higher timeout requirements will not be bid on.
     */
    maxJobExecutionTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
     * lower timeout requirements will not be bid on.
     */
    minJobExecutionTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
};

