# Authorization and authentication flow

Bacalhau authenticates and authorizes users in a multi-step flow.

## Requirements

We know our potential users have many possible requirements around auth and
exist across the entire spectrum from "no auth needed because its a simple local
deployment" to "enterprise-grade security for publicly accessible nodes". Hence,
the auth system needs to be unopinionated about how authentication and
authorization gets achieved.

The auth system has therefore been designed with a few goals in mind:

- **Flexible authentication**: it should be easy for users to add their own
  authentication method, including simple methods like using shared secrets and
  more complex methods up to OAuth and OIDC.
- **Flexible authorization**: it should be possible for users to be authorized
  based on a number of different modes, including group-based auth, RBAC and
  ABAC. The exact permissions of each should be customizable. The system should
  not require, for example, a particular model of "namespaces" or "workspaces"
  because these don't necessarily fit all use cases.
- **Future proofing**: the auth system should not require core-level upgrades
  to support advancements in cryptography. The hash functions and key sizes that
  are considered "secure" change over time, so the Bacalhau core should not be
  forced to have an opinion on this by the auth system and should not have to
  play "whack-a-mole" with supporting different configurations for different
  customers. Instead, it should be possible for customers to apply a policy that
  makes sense for them and upgrade security at their own pace.
- **Performance**: any calls to remote servers or complex algorithms to decide
  logic should happen once in the authentication process, and then subsequent
  calls to the API should introduce little overhead from authorization.

## Roles

- **Auth server** is a set of API endpoints that are trusted to make auth
  decisions. This is something built into the requester node and doesn't need to
  be a separate service, but could also be implemented as an external service if
  desired.
- **User agent** is a tool that acts on behalf of the user, running in a trusted
  way locally to them. The user agent submits API calls to the requester node on
  their behalf – so the CLI, Web UI and SDK are all user agents. We use the term
  "user agent" to differentiate from a "client", which in the OAuth sense means
  a third-party service that the user does not have complete trust in.

## Policies

Bacalhau implements flexible authentication and authorization using policies
which are written using a machine-executable policy format called Rego.

- Each **authentication policy** receives authentication credentials as input
  and outputs access tokens that will supplied to future API calls.
- Each **authorization policy** receives access tokens as input and outputs
  decisions about allowable access to APIs and job submission.

These two policies work together to define the entire authentication and
authorization scheme.

# Auth flow

The basic list of steps is:

1. Get the list of acceptable authn methods
2. Pick one and execute it, collecting any credentials from the user
3. Submit the credentials to the authn API
4. Receive an access token and use it in all future requests

## 1. Retrieve list of supported authentication methods

User agents make a request to their configured auth server to retrieve a list of
authentication methods, keyed by name.

```bash
curl -sL -X GET 'https://bootstrap.production.bacalhau.org/api/v1/auth'
```
```json
{
    "clientkey": {
        "type": "challenge",
        "params": {
            "nOnce": "9qn4v93qb4vq9hff",
            "minBits": 2048,
        },
    },
    "password": {
        "type": "ask",
        "params": {
            "$schema": ...
        },
    },
    "microsoft": {
        "type": "external",
        "params": {
            "base": "https://login.microsoft.com/?abc=...",
            "returnQueryParam": "redirect",
        },
    }
}
```

Each authentication method object describes:

* a type of authentication, identified by a specific key
* parameters to be used in running the authentication method, specific to that
  type

Each "type" can be used to implement a number of different authentication
methods. The types broadly correlate with behavior that the user agent needs to
take to run the authentication flow, such that there can be a single piece of
user agent code that is capable of running each type, with different input
parameters.

The supported types are:

### `challenge` authentication

This method is used to identify users via a private key that they hold. The
authentication response contains a `InputPhrase` that the user should sign and
return to the endpoint.

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "$id": "https://bacalhau.org/auth/challenge",
    "type": "object",
    "properties": {
        "InputPhrase": { "type": "string", "pattern": "[A-Za-z0-9]+" },
    }
}
```

### `ask` authentication

This method requires the user to manually input some information. This method
can be used to implement username and password authentication, shared secret
authentication, and even 2FA or security question auth.

The required information is represented by a JSON Schema in the object itself.
The implementation should parse the JSON Schema and ask the user questions to
populate an object that is valid by it.

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "$id": "https://bacalhau.org/auth/ask",
    "type": "object",
    "$ref": "https://json-schema.org/draft/2020-12/schema",
}
```

## 2. Run the authn flow and submit the result for an access token

The user agent decides which authentication method to use (e.g. by asking the
user, or by knowing it has an appropriate key) and operates the flow.

Once all the data for the method has been successfully collected, the user agent
POSTs the data to the auth endpoint for the method. The endpoint is the base
auth endpoint plus the name of the method, e.g. `/api/v1/auth/<method>`. So to
submit data for a "userpass" method, the user agent would POST to
`/api/v1/auth/userpass`.

## 3. Auth server checks the authn data against a policy

The auth server processes the request by inputting the auth credentials into a
auth policy. If the auth policy finds the passed data acceptable, it returns an
access token that the user can use in subsequent calls.

(Aside: there is actually no specification on the structure of the access token.
The user agent should treat it as an opaque blob that it receives from the auth
server and submits to the API server. Currently, all of the core Bacalhau code
also does not have any opinion of the auth token – it is not assumed to be any
specific type of object, and all parsing and handling is handled by the Rego
policies. However, all of the currently implemented Rego policies output and
expect JWTs, and it is recommended that users continue to use this convention.
The rest of this document will assume access tokens are JWTs.)

The signed JWT is returned to the user agent. The user agent takes appropriate
steps to keep the access token secret.

In principle, the auth policy can return any JWT it wishes, which will be
interpreted later in the API auth policy – it is up to the authn policy and the
authz policy to work together to apply auth. The policy to run is identified by
the `Node.Auth.Methods` variable, which is a map of method names to policy
paths.

However, the default authn and authz policies make decisions using namespaces.
Here, the authn policy returns a set of namespaces with associated access
permissions, and the authz policy controls access based on them.

In this default case, the JWT includes the fields:

### `iss` (issuer)

The node ID of the auth server.

### `sub` (subject)

A network-unique user ID, derived from the auth credentials. The `sub` does not
need to identify the same user across different authentication methods, but
should ideally be the same if the user logs in via the same auth method again.

### `ist` (issued at)

The timestamp when the token was issued.

### `exp` (expires at)

The timestamp after which the token is no longer valid.

### `ns` (namespaces)

A map of namespaces to permission bits.

The key in the map is a namespace name that the user has some level of access
of. Namespace names are ephemeral – i.e. there does not need to be a persistent
or coordinated store of namespaces shared across the whole cluster. Instead, the
**format** of namespace names is an interface for the network operator to
decide.

For example, the default policy will just give the user access to a namespace
identified by the `sub` field (e.g. their username). But in principle, more
complex setups involving groups could be used.

Namespace names can be a `*`, which by convention will match any set of
characters, like a filesystem glob. But it is up to the various auth policies to
actually implement this. So a JWT claim containing `"*"` would give default
permissions for all namespaces.

The value in the map is an unsigned integer encoding permission bits. If the
following bits are set:

- `0b00000001`: user can describe jobs in the namespace
- `0b00000010`: user can create jobs in the namespace
- `0b00000100`: user can download results from the namespace
- `0b00001000`: user can cancel jobs in the namespace

## 4. Make an API request and include the token

The user agent includes an `Authorization` header with the access token it
wishes to use passed as a bearer token:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX3459…
```

Note that the `Authorization` header is strictly optional – access for
unauthorized users is controlled using the policy, and may be allowed. The API
call is allowed to proceed if the authorization policy returns a positive
decision.

The requester node executes the API authorization policy and passes details of
the API call. The default policy is one where the namespaces of the token are
checked if present, and non-namespaced APIs require a valid signed token.

As above, custom policies are allowed. The policy to execute is defined by the
`Node.Auth.AccessPolicyPath` config variable. For non-namespaced APIs, such as
node APIs, the policy may make a blanket decision simply using whether the user
has an authorization token or not, or may choose to make a decision depending on
the type of authorization. For namespaced APIs, such as job APIs, the policy
should examine the namespaces in the JWT token and respond accordingly.

The authz server will return a `403 Forbidden` error if the user is not allowed
to carry out the requested action. It will also return a `401 Unauthorized`
error if the token the user passed is not valid for any future request. In the
latter case, the user agent should discard the token and execute the above flow
again to get a new one.

# Future work

There are a number of roadmap items that will enhance the auth system:

## Authn/z in the Web UI

The Web UI currently does not have any authn/z capability, and so can only work
with the default Bacalhau configuration which does not limit unauthenticated
users from querying read-only API endpoints.

To upgrade the Web UI to work in authenticated cases, it will be necessary to
implement the algorithms noted above. In short:

1. The Web UI will need to query the auth API endpoint for available authn
   methods.
2. It should then pick an appropriate authn method, either by asking the user,
   choosing based on known available data (e.g. existing presence of a private
   key), or by picking the only available option.
3. It should then run the authn flow for that type:
    - For `challenge` types, it will need a private key. It should probably
      generate and store one persistently rather than asking the user to upload
      theirs.
    - For `ask` types, it will need to parse the input JSON Schema and present a
      web form to collect the necessary authn credentials.
4. Once it has successfully authenticated, it should persistently store the
   access token and add it to all subsequent API requests.

## Addition of an `external` authentication type

This type will power future OAuth2/OIDC authentication. The principle is that:

1. The type will specify a remote endpoint to redirect the user to. The CLI will
   open a browser to this endpoint (or otherwise advise the user to do this) and
   the Web UI will just issue a redirect to this endpoint.

2. The user completes authentication at the remote service and is then
   redirected back to a supplied endpoint with valid credentials.

   The CLI may need to run a temporary web server to receive the redirect (this
   is how CLI tools like `gcloud` currently handle the OIDC flow). The Web UI
   will need to specify a redirect that it can subsequently decode credentials
   for.

   Also specified in the authentication method data will be any query
   parameters that the CLI/WebUI needs to populate with the redirect path. E.g.
   the specific OIDC scheme might specify the return location as a `?redirect`
   url query parameter, and the authentication type should specify the name of
   this parameter.

3. There doesn't need to be an optional step where the user exchanges the
   identity token they received from the remote auth server for a Bacalhau auth
   token. Instead, the system could just use the returned credential directly.

   However, this may be a beneficial step for mapping OIDC credentials into e.g.
   a JWT that specifies available namespaces. So there should probably be a step
   where the token received from the OIDC flow is passed to the authn method
   endpoint, and a policy has the chance to return a different token. In the
   basic case, it can check the validity of the token and return it unchanged.

4. The returned credential will be a JWT or similar access token. The user agent
   should use this credential to query the API as above. The authz policy should
   be configured to recognize these access tokens and apply authz control based
   on their content, as for the other methods.
