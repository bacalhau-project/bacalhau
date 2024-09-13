/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_AuthConfig } from './github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_AuthConfig';
import type { github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_UpdateConfig } from './github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_UpdateConfig';
import type { types_MetricsConfig } from './types_MetricsConfig';
import type { types_NodeConfig } from './types_NodeConfig';
import type { types_UserConfig } from './types_UserConfig';
export type types_BacalhauConfig = {
    auth?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_AuthConfig;
    /**
     * NB(forrest): this field shouldn't be persisted yet.
     */
    dataDir?: string;
    metrics?: types_MetricsConfig;
    node?: types_NodeConfig;
    update?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_UpdateConfig;
    user?: types_UserConfig;
};

