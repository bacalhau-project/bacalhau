/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_ClientTLSConfig } from './types_ClientTLSConfig';
import type { types_TLSConfiguration } from './types_TLSConfiguration';
export type types_APIConfig = {
    /**
     * ClientTLS specifies tls options for the client connecting to the
     * API.
     */
    clientTLS?: types_ClientTLSConfig;
    /**
     * Host is the hostname of an environment's public API servers.
     */
    host?: string;
    /**
     * Port is the port that an environment serves the public API on.
     */
    port?: number;
    /**
     * TLS returns information about how TLS is configured for the public server.
     * This is only used in APIConfig for NodeConfig.ServerAPI
     */
    tls?: types_TLSConfiguration;
};

