//go:build unit || !integration

package envelope

type TestPayload struct {
	Message string
	Value   int
}
