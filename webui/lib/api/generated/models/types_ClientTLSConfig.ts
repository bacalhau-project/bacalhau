/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type types_ClientTLSConfig = {
    /**
     * Used for NodeConfig.ClientAPI, specifies the location of a ca certificate
     * file (primarily for self-signed server certs). Will use HTTPS for requests.
     */
    cacert?: string;
    /**
     * Used for NodeConfig.ClientAPI, and when true instructs the client to use
     * HTTPS, but not to attempt to verify the certificate.
     */
    insecure?: boolean;
    /**
     * Used for NodeConfig.ClientAPI, instructs the client to connect over
     * TLS.  Auto enabled if Insecure or CACert are specified.
     */
    useTLS?: boolean;
};

