/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_SpecConfig } from './models_SpecConfig';
export type models_InputSource = {
    /**
     * Alias is an optional reference to this input source that can be used for
     * dynamic linking to this input. (e.g. dynamic import in wasm by alias)
     */
    Alias?: string;
    /**
     * Source is the source of the artifact to be downloaded, e.g a URL, S3 bucket, etc.
     */
    Source?: models_SpecConfig;
    /**
     * Target is the path where the artifact should be mounted on
     */
    Target?: string;
};

