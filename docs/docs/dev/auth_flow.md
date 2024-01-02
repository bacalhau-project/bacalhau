# Authorisation and authentication flow

Bacalhau authenticates and authorizes users in a multi-step flow.

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

Bacalhau implements flexible authentication and authorisation using policies
which are written using a machine-executable policy format called Rego.

- Each **authentication policy** receives authentication credentials as input
  and outputs JWT access tokens that will supplied to future API calls.
- Each **authorisation policy** receives access tokens as input and outputs
  decisions about allowable access to APIs and job submission.

These two policies work together to define the entire authentication and
authorisation scheme.

## 1. Retrieve list of supported authentication methods

User agents make a request to their configured auth server to retrieve a list of
authentication methods, keyed by name.

```bash
curl -sL -X GET 'https://bootstrap.production.bacalhau.org/api/v1/auth'
```
```json
{
    "privkey": {
        "challenge": {
            "nOnce": "9qn4v93qb4vq9hff",
            "minBits": 2048,
        },
        "tokenSchema": { ... },
    },
    "password": {
        "ask": {
            "$schema": ...
        },
        "tokenSchema": { ... },
    },
    "microsoft": {
        "login": {
            "base": "https://login.microsoft.com/?abc=...",
            "returnQueryParam": "redirect",
        },
        "tokenSchema": { ... },
    }
}
```

Each authentication method object describes:

* a type of authentication, identified by a specific key and value with
  type-specific data that allows the user agent to use the method
* a token schema, which is a `tokenSchema` key and JSON Schema value that a
  valid token will validate against

The token schema allows user agents to optionally decide whether the token they
have been issued is valid, and if not they can preemptively restart an
authentication flow. For example, if the token has expired, the validation for a
"issued at" field may fail, or if the requirements on key sizes have changed,
the user agent will know that it can no longer log in with that key.

### `challenge` authentication

This method is used to identify users via a private key that they hold. The
authentication response contains a `nOnce` that the user should sign and return
to the endpoint.

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "$id": "https://bacalhau.org/auth/challenge",
    "type": "object",
    "properties": {
        "nOnce": { "type": "string", "pattern": "[A-Za-z0-9]+" },
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

### `external` authentication

This method specifies an external endpoint to redirect the user to where they
can perform an OAuth-style flow. The user agent should prepare to receive a
redirect back to a URL it controls once the auth flow is complete.

The `base` property specifies a URL to send the user to. The `returnQueryParam`
names a URL query parameter: the user agent should URL-encode its return URL and
add it to the URL as the named parameter.

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "$id": "https://bacalhau.org/auth/external",
    "type": "object",
    "properties": {
        "base": { "type": "string" },
        "returnQueryParam": { "type": "string" },
    }
}
```

## 2. Check existing tokens against the schemas

The user agent checks whether any of the tokens it has are valid by matching
them against the JSON schemas returned by the list endpoint. If it has a valid
token, it can use that token in a future API call and skip steps 3 and 4.

The list endpoint response will have caching headers that should be used.

## 3. Run the authn flow and submit the result for an access token

The user agent decides which authentication method to use (e.g. by asking the
user, or by knowing it has an appropriate key) and operates the flow.

Once all the data for the method has been successfully collected, the user agent
POSTs the data to the auth endpoint for the method. The endpoint is the base
auth endpoint plus the name of the method, e.g. `/api/v1/auth/<method>`. So to
submit data for a "userpass" method, the user agent would POST to
`/api/v1/auth/userpass`.

The user agent includes an `Authorization` header with the access token it
wishes to use passed as a bearer token:

    Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9…

## 4. Auth server checks the authn data against a policy

The auth server processes the request by inputting the auth credentials into a
auth policy. The policy returns a set of namespaces with associated access
permissions, and the auth server encodes and signs the access into a JWT.

The signed JWT is returned to the user agent. The user agent takes appropriate
steps to keep the access token secret.

In principle, the auth policy can return any JWT it wishes, which will be
interpreted later in the API auth policy – it is up to the authn policy and the
authz policy to work together to apply auth. The policy to run is identified by
the `Node.Auth.Methods` variable, which is a map of method names to policy
paths.

However, the default policies will make decisions using namespaces. By default,
the JWT includes the fields:

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

The key in the map is an ephemeral namespace name that the user has some level
of access of. Namespace names are ephemeral – i.e. there does not need to be a
persistent or coordinated store of namespaces shared across the whole cluster.
Instead, the **format** of namespace names is an interface for the network
operator to decide.

Namespace names can contain a `*`, which by convention will match any set of
characters, like a filesystem glob. But it is up to the various auth policies to
actually implement this. So a JWT claim containing `"*"` would give default
permissions for all namespaces.

The value in the map is an unsigned integer encoding permission bits. If the
following bits are set:

- `0b00000001`: user can describe jobs in the namespace
- `0b00000010`: user can create jobs in the namespace
- `0b00000100`: user can download results from the namespace
- `0b00001000`: user can cancel jobs in the namespace

## 5. Make an API request and include the token

Once the User Agent has picked a token, it should include the token in the
`Authorization` HTTP header when it makes the request to call the API. Note that
the `Authorization` header is strictly optional – access for unauthorised users
is controlled using the policy.  The API call is allowed to proceed if the
authorisation policy returns a positive decision.

The requester node executes the API authorisation policy and passes details of
the API call. The default policy is one where the namespaces of the token are
checked if present, and non-namespaced APIs require a valid signed token.

As above, custom policies are allowed. The policy to execute is defined by the
`Node.Auth.AccessPolicyPath` config variable. For non-namespaced APIs, such as
node APIs, the policy may make a blanket decision simply using whether the user
has an authorisation token or not, or may choose to make a decision depending on
the type of authorisation. For namespaced APIs, such as job APIs, the policy
should examine the namespaces in the JWT token and respond accordingly.
