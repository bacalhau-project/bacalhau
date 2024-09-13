/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { apimodels_GetVersionResponse } from '../models/apimodels_GetVersionResponse';
import type { models_DebugInfo } from '../models/models_DebugInfo';
import type { models_NodeInfo } from '../models/models_NodeInfo';
import type { types_BacalhauConfig } from '../models/types_BacalhauConfig';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class OpsService {
    /**
     * @returns string OK
     * @throws ApiError
     */
    public static agentAlive(): CancelablePromise<string> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/agent/alive',
        });
    }
    /**
     * Returns the current configuration of the node.
     * @returns types_BacalhauConfig OK
     * @throws ApiError
     */
    public static agentConfig(): CancelablePromise<types_BacalhauConfig> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/agent/config',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns debug information on what the current node is doing.
     * @returns models_DebugInfo OK
     * @throws ApiError
     */
    public static agentDebug(): CancelablePromise<models_DebugInfo> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/agent/debug',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns the info of the node.
     * @returns models_NodeInfo OK
     * @throws ApiError
     */
    public static agentNode(): CancelablePromise<models_NodeInfo> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/agent/node',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * Returns the build version running on the server.
     * See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.
     * @returns apimodels_GetVersionResponse OK
     * @throws ApiError
     */
    public static agentVersion(): CancelablePromise<apimodels_GetVersionResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/agent/version',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
}
