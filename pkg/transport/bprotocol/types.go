package bprotocol

type Result[T any] struct {
	Response T
	Error    string
}
