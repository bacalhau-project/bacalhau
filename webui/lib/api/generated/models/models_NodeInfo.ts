/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_BuildVersionInfo } from './models_BuildVersionInfo';
import type { models_ComputeNodeInfo } from './models_ComputeNodeInfo';
import type { models_NodeType } from './models_NodeType';
export type models_NodeInfo = {
    BacalhauVersion?: models_BuildVersionInfo;
    ComputeNodeInfo?: models_ComputeNodeInfo;
    Labels?: Record<string, string>;
    /**
     * TODO replace all access on this field with the `ID()` method
     */
    NodeID?: string;
    NodeType?: models_NodeType;
};

