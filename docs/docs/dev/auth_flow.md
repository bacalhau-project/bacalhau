# Authorization and authentication flow

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

Bacalhau implements flexible authentication and authorization using policies
which are written using a machine-executable policy format called Rego.

- Each **authentication policy** receives authentication credentials as input
  and outputs JWT access tokens that will supplied to future API calls.
- Each **authorization policy** receives access tokens as input and outputs
  decisions about allowable access to APIs and job submission.

These two policies work together to define the entire authentication and
authorization scheme.

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
auth policy. If the auth policy finds the passed data acceptable, it returns a
signed JWT that the user can use as an access token.

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

The key in the map is an namespace name that the user has some level of access
of. Namespace names are ephemeral – i.e. there does not need to be a persistent
or coordinated store of namespaces shared across the whole cluster. Instead, the
**format** of namespace names is an interface for the network operator to
decide.

For example, the default policy will just give the user access to a namespace
identifier by the `sub` field (e.g. their username). But in principle, more
complex setups involving groups could be used.

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

## 4. Make an API request and include the token

The user agent includes an `Authorization` header with the access token it
wishes to use passed as a bearer token:

    Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX3459…

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
