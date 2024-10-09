#!bin/bashtub

 source bin/bacalhau.sh

 testcase_default_config_has_updates_enabled() {
     subject bacalhau config list --output=csv
     assert_equal 0 $status
     assert_not_equal $(echo "$stdout" | grep 'UpdateConfig.Interval' | cut -d, -f2) '0'
     assert_not_equal $(echo "$stdout" | grep 'UpdateConfig.Interval' | cut -d, -f2) ''
 }
