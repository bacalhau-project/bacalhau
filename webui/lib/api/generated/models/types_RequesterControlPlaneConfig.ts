/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_Duration } from './types_Duration';
export type types_RequesterControlPlaneConfig = {
    /**
     * This setting is the time period after which a compute node is considered to be unresponsive.
     * If the compute node misses two of these frequencies, it will be marked as unknown.  The compute
     * node should have a frequency setting less than this one to ensure that it does not keep
     * switching between unknown and active too frequently.
     */
    heartbeatCheckFrequency?: types_Duration;
    /**
     * This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
     */
    heartbeatTopic?: string;
    /**
     * This is the time period after which a compute node is considered to be disconnected. If the compute
     * node does not deliver a heartbeat every `NodeDisconnectedAfter` then it is considered disconnected.
     */
    nodeDisconnectedAfter?: types_Duration;
};

