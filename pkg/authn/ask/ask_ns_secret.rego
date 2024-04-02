package bacalhau.authn

import rego.v1

# Implements a policy where clients that supply a valid secret are
# permitted access. Anonymous users are not authenticated.
#
# Modify the `expected_secret` to control what secret is permitted access.
# Modify the `ns` key of the token to control what namespaces they can access.

schema := {
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"token": {"type": "string", "writeOnly": true},
	},
	"required": ["token"],
}

now := time.now_ns() / 1000

one_month := time.add_date(time.now_ns(), 0, 1, 0) / 1000

# expected_secret should be a random string issued to the user. in a simple live
# setup they can be hard coded here, and then apply appropriate file permissions
# to this policy.
expected_secret := "insert a secret string here"

valid_secret if {
	expected_secret == input.ask.token
}

token := t if {
	valid_secret
	t := io.jwt.encode_sign(
		{
			"typ": "JWT",
			"alg": "RS256",
		},
		{
			"iss": input.nodeId,
			"aud": [input.nodeId],
			"iat": now,
			"exp": one_month,
			"ns": {
				# Full access to all namespaces
				"*": full_access,
			},
		},
		input.signingKey,
	)
}

namespace_read := 1
namespace_write := 2
namespace_download := 4
namespace_cancel := 8

read_only := bits.or(namespace_read, namespace_download)
full_access := bits.or(bits.or(namespace_write, namespace_cancel), read_only)
