package bacalhau.authz
import rego.v1

default allow = false

# Anyone is able to query the auth endpoint to see what is suppoerted.
# This is the only anon-supported endpoint.
allow if {
    # TODO(forrest) [fixme] maybe make this more explicit.
    input.http.path[2] == "auth"
}

default token_valid = false

# If we managed to get namespaces, the token is valid
token_valid if {
    token_namespaces
}

# You are only allowed access if you have a token. Token == root
allow if {
    token_valid
}

# The list of namespaces from the verified access token
token_namespaces := ns if {
    authHeader := input.http.headers["Authorization"][0]
    startswith(authHeader, "Bearer ")
    accessToken := trim_prefix(authHeader, "Bearer ")

    [valid, header, claims] := io.jwt.decode_verify(accessToken, input.constraints)
    valid
    ns := claims["ns"]
}
