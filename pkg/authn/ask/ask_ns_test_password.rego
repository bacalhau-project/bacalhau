package bacalhau.authn

import rego.v1

# Implements a policy where clients that supply a valid username and password
# are permitted access. Anonymous users are not permitted.
#
# Modify the `userlist` to control what users are permitted access.
# Modify the `ns` key of the token to control what namespaces they can access.

schema := {
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"username": {"type": "string"},
		"password": {"type": "string", "writeOnly": true},
	},
	"required": ["username", "password"],
}

now := time.now_ns() / 1000

one_month := time.add_date(time.now_ns(), 0, 1, 0) / 1000

# userlist should be a map of usernames to scrypt-hashed passwords and salts. in
# a simple live setup they can be hard coded here as a map, and then apply
# appropriate file permissions to this policy.
userlist := {"username": [
	# hash corresponding to "password"
	"SN1U4DxjUhzyYG6p6nQ4by0IpudU8wdNs7Fpp42Ky9M=",
	# a randomly generated salt
	"d9ucnhE5kHEqm0YEWqN5qJmrHB+IDqjuPEwLkmZ9BGs=",
]}

valid_user := user if {
	some input.ask.username, _ in userlist

	user := input.ask.username
	hash := userlist[input.ask.username][0]
	salt := userlist[input.ask.username][1]
	hash == scrypt(input.ask.password, salt)
}

token := io.jwt.encode_sign(
	{
		"typ": "JWT",
		"alg": "RS256",
	},
	{
		"iss": input.nodeId,
		"sub": valid_user,
		"aud": [input.nodeId],
		"iat": now,
		"exp": one_month,
		"ns": {
			# Read-only access to all namespaces
			"*": read_only,
			# Writable access to own namespace
			valid_user: full_access,
		},
	},
	input.signingKey,
)

namespace_read := 1
namespace_write := 2
namespace_download := 4
namespace_cancel := 8

read_only := bits.or(namespace_read, namespace_download)
full_access := bits.or(bits.or(namespace_write, namespace_cancel), read_only)
