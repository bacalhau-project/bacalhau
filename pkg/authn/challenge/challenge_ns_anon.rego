package bacalhau.authn

import rego.v1

# Implements a policy where any clients with a client ID are permitted access.
# Anonymous users will authorised as long as they can generate a key pair.
#
# Modify the `ns` key of the token to control what namespaces they can access.

now := time.now_ns() / 1000

one_month := time.add_date(time.now_ns(), 0, 1, 0) / 1000

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

namespace_read     := 1
namespace_write    := 2
namespace_download := 4
namespace_cancel   := 8

read_only := bits.or(namespace_read, namespace_download)
full_access := bits.or(bits.or(namespace_write, namespace_cancel), read_only)
