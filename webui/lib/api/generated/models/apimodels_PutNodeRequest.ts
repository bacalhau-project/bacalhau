/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { apimodels_HTTPCredential } from './apimodels_HTTPCredential';
export type apimodels_PutNodeRequest = {
    Action?: string;
    Message?: string;
    NodeID?: string;
    credential?: apimodels_HTTPCredential;
    idempotencyToken?: string;
    namespace?: string;
};

