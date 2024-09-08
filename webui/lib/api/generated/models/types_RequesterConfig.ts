/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { models_FailureInjectionRequesterConfig } from './models_FailureInjectionRequesterConfig';
import type { models_JobSelectionPolicy } from './models_JobSelectionPolicy';
import type { types_DockerCacheConfig } from './types_DockerCacheConfig';
import type { types_Duration } from './types_Duration';
import type { types_EvaluationBrokerConfig } from './types_EvaluationBrokerConfig';
import type { types_JobDefaults } from './types_JobDefaults';
import type { types_JobStoreConfig } from './types_JobStoreConfig';
import type { types_RequesterControlPlaneConfig } from './types_RequesterControlPlaneConfig';
import type { types_SchedulerConfig } from './types_SchedulerConfig';
import type { types_StorageProviderConfig } from './types_StorageProviderConfig';
import type { types_WorkerConfig } from './types_WorkerConfig';
export type types_RequesterConfig = {
    controlPlaneSettings?: types_RequesterControlPlaneConfig;
    defaultPublisher?: string;
    evaluationBroker?: types_EvaluationBrokerConfig;
    /**
     * URL where to send external verification requests to.
     */
    externalVerifierHook?: string;
    failureInjectionConfig?: models_FailureInjectionRequesterConfig;
    housekeepingBackgroundTaskInterval?: types_Duration;
    jobDefaults?: types_JobDefaults;
    /**
     * How the node decides what jobs to run.
     */
    jobSelectionPolicy?: models_JobSelectionPolicy;
    jobStore?: types_JobStoreConfig;
    /**
     * ManualNodeApproval is a flag that determines if nodes should be manually approved or not.
     * By default, nodes are auto-approved to simplify upgrades, by setting this property to
     * true, nodes will need to be manually approved before they are included in node selection.
     */
    manualNodeApproval?: boolean;
    nodeInfoStoreTTL?: types_Duration;
    nodeRankRandomnessRange?: number;
    overAskForBidsFactor?: number;
    scheduler?: types_SchedulerConfig;
    storageProvider?: types_StorageProviderConfig;
    tagCache?: types_DockerCacheConfig;
    translationEnabled?: boolean;
    worker?: types_WorkerConfig;
};

