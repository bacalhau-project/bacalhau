/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_AuthConfig } from './types_AuthConfig';
import type { types_MetricsConfig } from './types_MetricsConfig';
import type { types_NodeConfig } from './types_NodeConfig';
import type { types_UpdateConfig } from './types_UpdateConfig';
import type { types_UserConfig } from './types_UserConfig';
export type types_BacalhauConfig = {
    auth?: types_AuthConfig;
    /**
     * NB(forrest): this field shouldn't be persisted yet.
     */
    dataDir?: string;
    metrics?: types_MetricsConfig;
    node?: types_NodeConfig;
    update?: types_UpdateConfig;
    user?: types_UserConfig;
};

