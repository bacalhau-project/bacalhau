#!bin/bashtub

source bin/bacalhau.sh

setup() {
    {
    bacalhau config set auth.methods "{\"Method\": \"shared_secret\", \"Policy\": {\"Type\": \"ask\", \"PolicyPath\": \"$ROOT/pkg/authn/ask/ask_ns_secret.rego\"}}"
    bacalhau config set auth.accesspolicypath "$ROOT/pkg/authz/policies/policy_ns_anon.rego"
    } >/dev/null

    create_node requester

    subject ls $BACALHAU_DIR/tokens.json
    assert_not_equal 0 $status
}

testcase_valid_token_is_accepted() {
    setup

    subject "echo insert a secret string here | bacalhau job list --output=json"
    assert_equal 0 $status
    assert_equal '[]' $stdout
}

testcase_invalid_token_is_rejected_and_not_persisted() {
    setup

    subject "echo not the secret string | bacalhau job list --output=json"
    assert_not_equal 0 $status
    assert_not_equal '[]' $stdout

    subject cat $BACALHAU_DIR/tokens.json
    assert_equal '' $stdout
}
