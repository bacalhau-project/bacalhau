//go:build integration

package devstack

//nolint:unused // golangci-lint complains that this is unused, but it's in pkg/test/devstack/sharding_test.go
func shouldRunShardingTest() (bool, error) {
	return true, nil
}
