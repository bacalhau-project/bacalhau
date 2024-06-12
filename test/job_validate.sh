#!bin/bashtub

source bin/bacalhau.sh

testcase_job_validate_valid_files() {
    for file in $ROOT/schemas/job/test_vectors/valid/*; do
        if [ -f "$file" ]; then
            subject bacalhau job validate "$file"
            assert_equal 0 $status
        fi
    done
}

testcase_job_validate_invalid_files() {
    for file in $ROOT/schemas/job/test_vectors/invalid/*; do
        if [ -f "$file" ]; then
            subject bacalhau job validate "$file"
            assert_equal 1 $status
        fi
    done
}
