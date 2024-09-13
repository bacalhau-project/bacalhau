/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class ComputeNodeService {
    /**
     * Returns debug information on what the current node is doing.
     * @returns string OK
     * @throws ApiError
     */
    public static apiServerDebug(): CancelablePromise<string> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/compute/debug',
            errors: {
                500: `Internal Server Error`,
            },
        });
    }
}
