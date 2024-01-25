# Authentication and authorization

Bacalhau includes a flexible auth system that supports multiple methods of auth
that are appropriate for different deployment environments.

## By default

With no specific authentication configuration supplied, Bacalhau runs in
"anonymous mode" – which allows unidentified users limited control over the
system. "Anonymous mode" is only appropriate for testing or evaluation setups.

In anonymous mode, Bacalhau will allow:

- Users identified by a self-generated private key to submit any job, cancel
  their own jobs, read job lists and describing jobs.
- Users not identified by any key to access other read-only endpoints, such as
  node or agent information.

## Restricting anonymous access

Bacalhau auth is controlled by policies. Configuring the auth system is done by
supplying a different policy file.

Restricting API access to only users that have authenticated requires specifying
a new **authorization policy**. You can download a policy that restricts
anonymous access and install it by using:

    curl -sL https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/pkg/authz/policies/policy_ns_anon.rego -o ~/.bacalhau/no-anon.rego
    bacalhau config set Node.Auth.AccessPolicyPath ~/.bacalhau/no-anon.rego

Once the node is restarted, accessing the node APIs will require the user to be
authenticated, but by default will still allow users with a self-generated key
to authenticate themselves.

Restricting the list of keys that can authenticate to only a known set requires
specifying a new **authentication policy**. You can download a policy that
restricts key-based access and install it by using:

    curl -sL https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/pkg/authn/challenge/challenge_ns_no_anon.rego -o ~/.bacalhau/challenge_ns_no_anon.rego
    bacalhau config set Node.Auth.Methods.ClientKey.Type challenge
    bacalhau config set Node.Auth.Methods.ClientKey.PolicyPath ~/.bacalhau/challenge_ns_no_anon.rego

Then, modify the `allowed_clients` variable in `challange_ns_no_anon.rego` to
include acceptable client IDs, found by running `bacalhau id`.

    bacalhau id | jq -rc .ClientID

Once the node is restarted, only keys in the allowed list will be able to access
any API.
