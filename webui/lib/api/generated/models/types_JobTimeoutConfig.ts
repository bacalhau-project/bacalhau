/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_Duration } from './types_Duration';
export type types_JobTimeoutConfig = {
    /**
     * DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
     * no timeout requirement defined.
     */
    defaultJobExecutionTimeout?: types_Duration;
    /**
     * JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
     * check.
     */
    jobExecutionTimeoutClientIDBypassList?: Array<string>;
    /**
     * JobNegotiationTimeout default timeout value to hold a bid for a job
     */
    jobNegotiationTimeout?: types_Duration;
    /**
     * MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
     * higher timeout requirements will not be bid on.
     */
    maxJobExecutionTimeout?: types_Duration;
    /**
     * MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
     * lower timeout requirements will not be bid on.
     */
    minJobExecutionTimeout?: types_Duration;
};

