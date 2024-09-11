/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration } from './github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration';
import type { logger_LogMode } from './logger_LogMode';
import type { types_APIConfig } from './types_APIConfig';
import type { types_ComputeConfig } from './types_ComputeConfig';
import type { types_FeatureConfig } from './types_FeatureConfig';
import type { types_IpfsConfig } from './types_IpfsConfig';
import type { types_NetworkConfig } from './types_NetworkConfig';
import type { types_RequesterConfig } from './types_RequesterConfig';
import type { types_WebUIConfig } from './types_WebUIConfig';
export type types_NodeConfig = {
    /**
     * AllowListedLocalPaths contains local paths that are allowed to be mounted into jobs
     */
    allowListedLocalPaths?: Array<string>;
    clientAPI?: types_APIConfig;
    compute?: types_ComputeConfig;
    /**
     * TODO(forrest) [refactor]: rename this to ExecutorStoragePath
     * Deprecated: replaced by cfg.ComputeDir()
     */
    computeStoragePath?: string;
    /**
     * What features should not be enabled even if installed
     */
    disabledFeatures?: types_FeatureConfig;
    downloadURLRequestRetries?: number;
    downloadURLRequestTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * Deprecated: replaced by cfg.PluginsDir()
     */
    executorPluginPath?: string;
    ipfs?: types_IpfsConfig;
    /**
     * Labels to apply to the node that can be used for node selection and filtering
     */
    labels?: Record<string, string>;
    loggingMode?: logger_LogMode;
    name?: string;
    nameProvider?: string;
    network?: types_NetworkConfig;
    requester?: types_RequesterConfig;
    serverAPI?: types_APIConfig;
    strictVersionMatch?: boolean;
    /**
     * Type is "compute", "requester" or both
     */
    type?: Array<string>;
    volumeSizeRequestTimeout?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * Configuration for the web UI
     */
    webUI?: types_WebUIConfig;
};

