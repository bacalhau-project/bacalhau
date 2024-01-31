package models

type ExecutionLogType int

const (
	executionLogTypeUnknown ExecutionLogType = iota
	ExecutionLogTypeSTDOUT
	ExecutionLogTypeSTDERR
)

type ExecutionLog struct {
	Type ExecutionLogType
	Line string
}
