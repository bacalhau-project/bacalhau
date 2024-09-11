/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration } from './github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration';
export type types_ComputeControlPlaneConfig = {
    /**
     * How often the compute node will send a heartbeat to the requester node to let it know
     * that the compute node is still alive. This should be less than the requester's configured
     * heartbeat timeout to avoid flapping.
     */
    heartbeatFrequency?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
     */
    heartbeatTopic?: string;
    /**
     * The frequency with which the compute node will send node info (inc current labels)
     * to the controlling requester node.
     */
    infoUpdateFrequency?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
    /**
     * How often the compute node will send current resource availability to the requester node.
     */
    resourceUpdateFrequency?: github_com_bacalhau_project_bacalhau_pkg_config_legacy_types_Duration;
};

