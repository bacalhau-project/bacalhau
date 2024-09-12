/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_NetworkClusterConfig } from './types_NetworkClusterConfig';
export type types_NetworkConfig = {
    advertisedAddress?: string;
    authSecret?: string;
    cluster?: types_NetworkClusterConfig;
    orchestrators?: Array<string>;
    port?: number;
    storeDir?: string;
};

