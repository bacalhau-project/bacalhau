//go:build unit || !integration

package watcher

type TestObject struct {
	Name  string
	Value int
}
