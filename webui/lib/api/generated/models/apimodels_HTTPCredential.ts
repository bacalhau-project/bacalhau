/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type apimodels_HTTPCredential = {
    /**
     * For authorization schemes that provide multiple values, a map of names to
     * values providing the credential
     */
    params?: Record<string, string>;
    /**
     * An HTTP authorization scheme, such as one registered with IANA
     * https://www.iana.org/assignments/http-authschemes/http-authschemes.xhtml
     */
    scheme?: string;
    /**
     * For authorization schemes that only provide a single value, such as
     * Basic, the single string value providing the credential
     */
    value?: string;
};

