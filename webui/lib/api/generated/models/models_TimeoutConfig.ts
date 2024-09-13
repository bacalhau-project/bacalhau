/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type models_TimeoutConfig = {
    /**
     * ExecutionTimeout is the maximum amount of time a task is allowed to run in seconds.
     * Zero means no timeout, such as for a daemon task.
     */
    ExecutionTimeout?: number;
    /**
     * QueueTimeout is the maximum amount of time a task is allowed to wait in the orchestrator
     * queue in seconds before being scheduled. Zero means no timeout.
     */
    QueueTimeout?: number;
    /**
     * TotalTimeout is the maximum amount of time a task is allowed to complete in seconds.
     * This includes the time spent in the queue, the time spent executing and the time spent retrying.
     * Zero means no timeout.
     */
    TotalTimeout?: number;
};

