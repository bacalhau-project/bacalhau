/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_JobSelectionDataLocality } from './models_JobSelectionDataLocality';
export type models_JobSelectionPolicy = {
    /**
     * should we accept jobs that specify networking
     * the default is "reject"
     */
    accept_networked_jobs?: boolean;
    /**
     * this describes if we should run a job based on
     * where the data is located - i.e. if the data is "local"
     * or if the data is "anywhere"
     */
    locality?: models_JobSelectionDataLocality;
    probe_exec?: string;
    /**
     * external hooks that decide if we should take on the job or not
     * if either of these are given they will override the data locality settings
     */
    probe_http?: string;
    /**
     * should we reject jobs that don't specify any data
     * the default is "accept"
     */
    reject_stateless_jobs?: boolean;
};

