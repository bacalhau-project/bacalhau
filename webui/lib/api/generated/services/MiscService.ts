/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { shared_VersionRequest } from '../models/shared_VersionRequest';
import type { shared_VersionResponse } from '../models/shared_VersionResponse';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class MiscService {
    /**
     * Returns the build version running on the server.
     * See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.
     * @param versionRequest Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
     * @returns shared_VersionResponse OK
     * @throws ApiError
     */
    public static apiServerVersion(
        versionRequest: shared_VersionRequest,
    ): CancelablePromise<shared_VersionResponse> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/version',
            body: versionRequest,
            errors: {
                400: `Bad Request`,
                500: `Internal Server Error`,
            },
        });
    }
}
