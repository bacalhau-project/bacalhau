# Authentication and authorization

Bacalhau includes a flexible auth system that supports multiple methods of auth
that are appropriate for different deployment environments.

## By default

With no specific authentication configuration supplied, Bacalhau runs in
"anonymous mode" – which allows unidentified users limited control over the
system. "Anonymous mode" is only appropriate for testing or evaluation setups.

In anonymous mode, Bacalhau will allow:

- Users identified by a self-generated private key to submit any job and cancel
  their own jobs
- Users not identified by any key to access read-only endpoints, such as job
  lists and `bacalhau describe`.

## Restricting anonymous access

Bacalhau auth is controlled by policies. Configuring the auth system is done by
just supplying a different policy file.

Restricting API access to only users that have authenticated requires specifying
a new **authorisation policy**. You can download a policy that restricts
anonymous access and install it by using:

    curl -sL https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/pkg/authz/no-anon.opa -o ~/.bacalhau/no-anon.opa
    bacalhau config set Node.Auth.AccessPolicyPath ~/.bacalhau/no-anon.opa

Once the node is restarted, accessing the node APIs will require the user to be
authenticated, but by default will still allow users with a self-generated key
to authenticate themselves.

Restricting the list of keys that can authenticate to only a known set requires
specifying a new **authentication policy**. You can download a policy that
restricts key-based access and install it by using:

    curl -sL https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/pkg/authn/keyring.opa -o ~/.bacalhau/keyring.opa
    bacalhau config set Node.Auth.Methods.PrivateKey ~/.bacalhau/keyring.opa

Then, modify the `allowed_ids` variable in `keyring.opa` to include acceptable
client IDs, found by running `bacalhau id`.

    bacalhau id | jq -rc .ClientID

Once the node is restarted, only keys in the allowed list will be able to access
any API.

## Username and password access

Users can authenticate using a username and password instead of specifying a
private key for access. Again, this just requires installation of an appropriate
policy.

    curl -sL https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/pkg/authn/userpass.opa -o ~/.bacalhau/userpass.opa
    bacalhau config set Node.Auth.Methods.Password ~/.bacalhau/userpass.opa

Passwords are not stored in plaintext and are salted. To generate a salted
password, a code snippet like this will ask for a password on the terminal and
salt it using the node's unique ID:

    python3 -c "import getpass;import bcrypt;print(bcrypt.kdf(password=getpass.getpass().encode(),salt='$(bacalhau id | jq -rc .ID)'.encode(),desired_key_bytes=32,rounds=100).hex())"

Then, add the username and salted password into the `userpass.opa`.
