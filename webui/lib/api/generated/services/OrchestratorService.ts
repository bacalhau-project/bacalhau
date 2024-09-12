/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { apimodels_GetJobResponse } from '../models/apimodels_GetJobResponse';
import type { apimodels_GetNodeResponse } from '../models/apimodels_GetNodeResponse';
import type { apimodels_ListJobExecutionsResponse } from '../models/apimodels_ListJobExecutionsResponse';
import type { apimodels_ListJobHistoryResponse } from '../models/apimodels_ListJobHistoryResponse';
import type { apimodels_ListJobResultsResponse } from '../models/apimodels_ListJobResultsResponse';
import type { apimodels_ListJobsResponse } from '../models/apimodels_ListJobsResponse';
import type { apimodels_ListNodesResponse } from '../models/apimodels_ListNodesResponse';
import type { apimodels_PutJobRequest } from '../models/apimodels_PutJobRequest';
import type { apimodels_PutJobResponse } from '../models/apimodels_PutJobResponse';
import type { apimodels_PutNodeRequest } from '../models/apimodels_PutNodeRequest';
import type { apimodels_PutNodeResponse } from '../models/apimodels_PutNodeResponse';
import type { apimodels_StopJobResponse } from '../models/apimodels_StopJobResponse';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class OrchestratorService {
    /**
     * Returns a list of jobs.
     * Returns a list of jobs.
     * @param namespace Namespace to get the jobs for
     * @param limit Limit the number of jobs returned
     * @param nextToken Token to get the next page of jobs
     * @param reverse Reverse the order of the jobs
     * @param orderBy Order the jobs by the given field
     * @returns apimodels_ListJobsResponse OK
     * @throws ApiError
     */
    public static orchestratorListJobs(
        namespace?: string,
        limit?: number,
        nextToken?: string,
        reverse?: boolean,
        orderBy?: string,
    ): CancelablePromise<apimodels_ListJobsResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs',
            query: {
                'namespace': namespace,
                'limit': limit,
                'next_token': nextToken,
                'reverse': reverse,
                'order_by': orderBy,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Submits a job to the orchestrator.
     * Submits a job to the orchestrator.
     * @param putJobRequest Job to submit
     * @returns apimodels_PutJobResponse OK
     * @throws ApiError
     */
    public static orchestratorPutJob(
        putJobRequest: apimodels_PutJobRequest,
    ): CancelablePromise<apimodels_PutJobResponse> {
        return __request(OpenAPI, {
            method: 'PUT',
            url: '/api/v1/orchestrator/jobs',
            body: putJobRequest,
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns a job.
     * Returns a job.
     * @param id ID to get the job for
     * @param include Takes history and executions as options. If empty will not include anything else.
     * @param limit Number of history or executions to fetch. Should be used in conjugation with include
     * @returns apimodels_GetJobResponse OK
     * @throws ApiError
     */
    public static orchestratorGetJob(
        id: string,
        include?: string,
        limit?: number,
    ): CancelablePromise<apimodels_GetJobResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs/{id}',
            path: {
                'id': id,
            },
            query: {
                'include': include,
                'limit': limit,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Stops a job.
     * Stops a job.
     * @param id ID to stop the job for
     * @param reason Reason for stopping the job
     * @returns apimodels_StopJobResponse OK
     * @throws ApiError
     */
    public static orchestratorStopJob(
        id: string,
        reason?: string,
    ): CancelablePromise<apimodels_StopJobResponse> {
        return __request(OpenAPI, {
            method: 'DELETE',
            url: '/api/v1/orchestrator/jobs/{id}',
            path: {
                'id': id,
            },
            query: {
                'reason': reason,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns the executions of a job.
     * Returns the executions of a job.
     * @param id ID to get the job executions for
     * @param orderBy Order the executions by the given field
     * @param namespace Namespace to get the jobs for
     * @param limit Limit the number of executions returned
     * @param nextToken Token to get the next page of executions
     * @param reverse Reverse the order of the executions
     * @returns apimodels_ListJobExecutionsResponse OK
     * @throws ApiError
     */
    public static orchestratorJobExecutions(
        id: string,
        orderBy: string,
        namespace?: string,
        limit?: number,
        nextToken?: string,
        reverse?: boolean,
    ): CancelablePromise<apimodels_ListJobExecutionsResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs/{id}/executions',
            path: {
                'id': id,
            },
            query: {
                'namespace': namespace,
                'limit': limit,
                'next_token': nextToken,
                'reverse': reverse,
                'order_by': orderBy,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns the history of a job.
     * Returns the history of a job.
     * @param id ID to get the job history for
     * @param since Only return history since this time
     * @param eventType Only return history of this event type
     * @param executionId Only return history of this execution ID
     * @param nextToken Token to get the next page of the jobs
     * @returns apimodels_ListJobHistoryResponse OK
     * @throws ApiError
     */
    public static orchestratorJobHistory(
        id: string,
        since?: string,
        eventType?: string,
        executionId?: string,
        nextToken?: string,
    ): CancelablePromise<apimodels_ListJobHistoryResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs/{id}/history',
            path: {
                'id': id,
            },
            query: {
                'since': since,
                'event_type': eventType,
                'execution_id': executionId,
                'next_token': nextToken,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Streams the logs for a current job/execution via WebSocket
     * Establishes a WebSocket connection to stream output from the job specified by `id`
     * The stream will continue until either the client disconnects or the execution completes
     * @param id ID of the job to stream logs for
     * @param executionId Fetch logs for a specific execution
     * @param tail Fetch historical logs
     * @param follow Follow the logs
     * @returns void
     * @throws ApiError
     */
    public static orchestratorLogs(
        id: string,
        executionId?: string,
        tail?: boolean,
        follow?: boolean,
    ): CancelablePromise<void> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs/{id}/logs',
            path: {
                'id': id,
            },
            query: {
                'execution_id': executionId,
                'tail': tail,
                'follow': follow,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns the results of a job.
     * Returns the results of a job.
     * @param id ID to get the job results for
     * @returns apimodels_ListJobResultsResponse OK
     * @throws ApiError
     */
    public static orchestratorJobResults(
        id: string,
    ): CancelablePromise<apimodels_ListJobResultsResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/jobs/{id}/results',
            path: {
                'id': id,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns a list of orchestrator nodes.
     * Returns a list of orchestrator nodes.
     * @param limit Limit the number of node returned
     * @param nextToken Token to get the next page of nodes
     * @param reverse Reverse the order of the nodes
     * @param orderBy Order the nodes by given field
     * @param filterApproval Filter Approval
     * @param filterStatus Filter Status
     * @returns apimodels_ListNodesResponse OK
     * @throws ApiError
     */
    public static orchestratorListNodes(
        limit?: number,
        nextToken?: string,
        reverse?: boolean,
        orderBy?: string,
        filterApproval?: string,
        filterStatus?: string,
    ): CancelablePromise<apimodels_ListNodesResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/nodes',
            query: {
                'limit': limit,
                'next_token': nextToken,
                'reverse': reverse,
                'order_by': orderBy,
                'filter_approval': filterApproval,
                'filter-status': filterStatus,
            },
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Update an orchestrator node.
     * Update an orchestrator node.
     * @param id ID of the orchestrator node.
     * @param putNodeRequest Put Node Request
     * @returns apimodels_PutNodeResponse OK
     * @throws ApiError
     */
    public static orchestratorUpdateNode(
        id: string,
        putNodeRequest: apimodels_PutNodeRequest,
    ): CancelablePromise<apimodels_PutNodeResponse> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/orchestrator/nodes',
            path: {
                'id': id,
            },
            body: putNodeRequest,
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Get an orchestrator node
     * Get an orchestrator node
     * @param id ID of the orchestrator node to fetch for.
     * @returns apimodels_GetNodeResponse OK
     * @throws ApiError
     */
    public static orchestratorGetNode(
        id: string,
    ): CancelablePromise<apimodels_GetNodeResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/orchestrator/nodes/{id}',
            path: {
                'id': id,
            },
            errors: {
                400: `Bad Request`,
                404: `Not Found`,
                500: `Internal Server Error`,
            },
        });
    }
}
