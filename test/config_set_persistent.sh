#!bin/bashtub

source bin/bacalhau.sh


testcase_config_set_is_persistent() {
   TEST_VALUE=$RANDOM
   subject bacalhau config set --config=./test-persistent.yaml 'NameProvider' $TEST_VALUE
   assert_equal 0 $status

   subject file ./test-persistent.yaml
   assert_equal 0 $status

   # Verify the contents of the config file
   subject cat "./test-persistent.yaml | grep -i NameProvider"
   assert_match "${TEST_VALUE}" "$stdout"
   rm ./test-persistent.yaml
}




