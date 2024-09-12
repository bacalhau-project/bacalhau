/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type types_TLSConfiguration = {
    /**
     * AutoCert specifies a hostname for a certificate to be obtained via ACME.
     * This is only used by the server, and only by the requester node when it
     * has a publicly resolvable domain name.
     */
    autoCert?: string;
    /**
     * AutoCertCachePath specifies the directory where the autocert process
     * will cache certificates to avoid rate limits.
     */
    autoCertCachePath?: string;
    /**
     * SelfSignedCert will auto-generate a self-signed certificate for the
     * requester node if TLS certificates have not been provided.
     */
    selfSigned?: boolean;
    /**
     * ServerCertificate specifies the location of a TLS certificate to be used
     * by the requester to serve TLS requests
     */
    serverCertificate?: string;
    /**
     * ServerKey is the TLS server key to match the certificate to allow the
     * requester to server TLS.
     */
    serverKey?: string;
};

