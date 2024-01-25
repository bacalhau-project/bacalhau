package bacalhau.authz
import rego.v1

default allow = false

allow if {
    input.http.method == "GET"
    input.http.path == ["api", "v1", "hello"]
}
