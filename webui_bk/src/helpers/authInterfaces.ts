import { JSONSchema7 } from "json-schema";
import { ListRequest, ListResponse } from "./baseInterfaces";

// A request to list the authentication methods that the node supports.
export interface ListAuthnMethodsRequest extends ListRequest { }

// A response listing the authentication methods that the node supports.
export interface ListAuthnMethodsResponse extends ListResponse {
    // The name of a method mapped to that method's requirements.
    // The name must be subsequently used to submit data.
    Methods: {[key: string]: Requirement}
}

// A requirement of an authn method, with the params varying per-type.
export type Requirement = {
    type: "challenge"
    params: ChallengeRequirement
} | {
    type: "ask"
    params: AskRequirement
}

// The "ask" type just gives the client a JSON Schema that describes the fields
// it needs to collect from the user and submit.
export type AskRequirement = JSONSchema7

// The "challenge" type gives the client an input phrase it must sign using its
// private key.
export interface ChallengeRequirement {
    InputPhrase: string
}

// A request to authenticate using a given method, including any credentials.
export interface AuthnRequest {
    Name: string
    MethodData: AskRequest | ChallengeRequest
}

// The "ask" type needs the fields requested from the user by the JSON Schema.
export type AskRequest = {[key: string]: string}

// The "challenge" type needs the signature of the phrase and the associated
// public key.
export interface ChallengeRequest {
    PhraseSignature: string
    PublicKey: string
}

export interface AuthnResponse {
    Authentication: Authentication
}

// A response from trying to authenticate.
export interface Authentication {
    // Whether the authentication was successful.
    success: boolean
    // Any additional info about why authentication was successful or not.
    reason?: string
    // The token the client should use in subsequent API requests.
    token?: string
}
