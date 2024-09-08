/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { types_AuthenticatorConfig } from './types_AuthenticatorConfig';
export type types_AuthConfig = {
    /**
     * AccessPolicyPath is the path to a file or directory that will be loaded as
     * the policy to apply to all inbound API requests. If unspecified, a policy
     * that permits access to all API endpoints to both authenticated and
     * unauthenticated users (the default as of v1.2.0) will be used.
     */
    accessPolicyPath?: string;
    /**
     * Methods maps "method names" to authenticator implementations. A method
     * name is a human-readable string chosen by the person configuring the
     * system that is shown to users to help them pick the authentication method
     * they want to use. There can be multiple usages of the same Authenticator
     * *type* but with different configs and parameters, each identified with a
     * unique method name.
     *
     * For example, if an implementation wants to allow users to log in with
     * Github or Bitbucket, they might both use an authenticator implementation
     * of type "oidc", and each would appear once on this provider with key /
     * method name "github" and "bitbucket".
     *
     * By default, only a single authentication method that accepts
     * authentication via client keys will be enabled.
     */
    methods?: Record<string, types_AuthenticatorConfig>;
    /**
     * TokensPath is the location where a state file of tokens will be stored.
     * By default it will be local to the Bacalhau repo, but can be any location
     * in the host filesystem. Tokens are sensitive and should be stored in a
     * location that is only readable to the current user.
     * Deprecated: replaced by cfg.AuthTokensPath()
     */
    tokensPath?: string;
};

