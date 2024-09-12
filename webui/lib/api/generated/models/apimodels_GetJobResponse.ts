/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { apimodels_ListJobExecutionsResponse } from './apimodels_ListJobExecutionsResponse';
import type { apimodels_ListJobHistoryResponse } from './apimodels_ListJobHistoryResponse';
import type { models_Job } from './models_Job';
export type apimodels_GetJobResponse = {
    Executions?: apimodels_ListJobExecutionsResponse;
    History?: apimodels_ListJobHistoryResponse;
    Job?: models_Job;
};

