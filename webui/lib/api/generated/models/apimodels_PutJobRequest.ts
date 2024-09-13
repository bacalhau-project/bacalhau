/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { apimodels_HTTPCredential } from './apimodels_HTTPCredential';
import type { models_Job } from './models_Job';
export type apimodels_PutJobRequest = {
    Job?: models_Job;
    credential?: apimodels_HTTPCredential;
    idempotencyToken?: string;
    namespace?: string;
};

