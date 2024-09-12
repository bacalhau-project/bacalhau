/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_NodeInfo } from '../models/models_NodeInfo';
import type { types_HealthInfo } from '../models/types_HealthInfo';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class UtilsService {
    /**
     * @returns string OK
     * @throws ApiError
     */
    public static home(): CancelablePromise<string> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/',
        });
    }
    /**
     * @returns types_HealthInfo OK
     * @throws ApiError
     */
    public static healthz(): CancelablePromise<types_HealthInfo> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/healthz',
        });
    }
    /**
     * Returns the id of the host node.
     * @returns string OK
     * @throws ApiError
     */
    public static id(): CancelablePromise<string> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/id',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
    /**
     * @returns string TODO
     * @throws ApiError
     */
    public static livez(): CancelablePromise<string> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/livez',
        });
    }
    /**
     * Returns the info of the node.
     * @returns models_NodeInfo OK
     * @throws ApiError
     */
    public static nodeInfo(): CancelablePromise<models_NodeInfo> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/node_info',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
}
