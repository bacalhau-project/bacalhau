#!bin/bashtub

source bin/bacalhau.sh

setup() {
    {
    bacalhau config set auth.methods "{\"Method\": \"shared_secret\", \"Policy\": {\"Type\": \"ask\", \"PolicyPath\": \"$ROOT/pkg/authn/ask/ask_ns_secret.rego\"}}"
    bacalhau config set auth.accesspolicypath "$ROOT/pkg/authz/policies/policy_ns_anon.rego"
    } >/dev/null

    subject 'bacalhau config list | grep auth.methods'
    assert_match 'shared_secret' $stdout

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

testcase_token_works_for_v1_api() {
    setup

    subject "echo insert a secret string here | bacalhau list --output=json"
    assert_equal 0 $status
    assert_equal '[]' $stdout
}

testcase_input_can_be_cancelled() {
    setup

    # Run bacalhau job list in a screen session, else the auth code will detect
    # that a TTY is not in use and won't use the hanging code
    CMD='bacalhau job list --output=json'
    rm -f screenlog.0
    screen -L -d -m bacalhau job list --output=json
    while ! grep 'token:' screenlog.0 1>/dev/null 2>&1; do sleep 0.01; done

    PIDS=$(pgrep -fx $CMD)
    assert_equal 0 $?
    assert_not_equal '' $PIDS

    # Send 'job list' an interrupt signal as if the user pressed Ctrl+C
    kill -INT $PIDS
    sleep 1

    NOW=$(pgrep -fx $CMD)
    assert_not_equal 0 $?
    assert_equal '' $NOW

    assert_not_match '[]' "$(cat screenlog.0)"
    assert_match 'context canceled' "$(cat screenlog.0)"

    rm screenlog.0
    pkill -9 "$CMD"
}

testcase_invalid_token_is_rejected_and_not_persisted() {
    setup

    subject "echo not the secret string | bacalhau job list --output=json"
    assert_not_equal 0 $status
    assert_not_equal '[]' $stdout

    subject cat $BACALHAU_DIR/tokens.json
    assert_equal '{}' $stdout
}

testcase_invalid_or_expired_token_is_removed() {
    setup
    testcase_valid_token_is_accepted

    subject cat $BACALHAU_DIR/tokens.json
    assert_not_equal '' $stdout

    INVALID_TOKEN='eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNTE2MjM5MDIyLCJpYXQiOjE1MTYyMzkwMjIsIm5zIjp7IioiOjd9fQ.lNkaXliBT2wmnG95ZnI2jq7P4SEgBQlb23XeSa9Ds9DRjHDYi8-Bt99vmh8TVGCx9RMBLu8b1EXRcZI0iuR7XwuCXmRaW9vKjDClpHSnkSyQL3vWpw9zVKIf3OLt5dsqXkxOn3BfUwf3XmMqWfrrDM8MGGk5Zn5jPP3O0eMjFKteI8mwXjaZJlOnmbIjrKAeg9-Vvyswgme4OFioM9g1Lgz81tqoZzJWMmx6uwwM2Uv1FLV9bHhUcl4eRr3Es_SBOvRkVoU_cIh24TitxD_kBabof_PCjvpdzVkuZzOMvD3BNuvgpO6tbRJaqnPl9iks65Fn0Q56mC33Q18ijVw5bQ'
    jq -rc "{(keys[0]): \"$INVALID_TOKEN\"}" <$BACALHAU_DIR/tokens.json >$BACALHAU_DIR/tokens.temp
    mv $BACALHAU_DIR/tokens.temp $BACALHAU_DIR/tokens.json

    subject 'bacalhau job list --output=json </dev/null'
    assert_not_equal 0 $status
    assert_not_equal '[]' $stdout

    subject cat $BACALHAU_DIR/tokens.json
    assert_equal '{}' $stdout
}
