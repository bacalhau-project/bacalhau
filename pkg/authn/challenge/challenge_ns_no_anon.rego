package bacalhau.authn

import rego.v1

# Implements a policy where only clients with client IDs that match a known list
# are permitted access. Anonymous users will not be authenticated.
#
# Add client IDs into `allowed_clients` to configure access.
# Modify the `ns` key of the token to control what namespaces they can access.

now := time.now_ns() / 1000

one_month := time.add_date(time.now_ns(), 0, 1, 0) / 1000

allowed_clients := [
    # Insert client IDs here
]

token := token if {
    input.clientId in allowed_clients

    token := io.jwt.encode_sign(
	{
		"typ": "JWT",
		"alg": "RS256",
	},
	{
		"iss": input.nodeId,
		"sub": input.clientId,
		"aud": [input.nodeId],
		"iat": now,
		"exp": one_month,
		"ns": {
			# Read-only access to all namespaces
			"*": read_only,
			# Writable access to own namespace
			input.clientId: full_access,
		},
	},
	input.signingKey,
)
}

namespace_read     := 1
namespace_write    := 2
namespace_download := 4
namespace_cancel   := 8

read_only := bits.or(namespace_read, namespace_download)
full_access := bits.or(bits.or(namespace_write, namespace_cancel), read_only)
