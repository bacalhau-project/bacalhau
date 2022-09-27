package runtime

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Event struct {
	*StartEvent      `json:"start_event,omitempty"`
	*MessageEvent    `json:"message_event,omitempty"`
	*SuccessEvent    `json:"success_event,omitempty"`
	*FailureEvent    `json:"failure_event,omitempty"`
	*CrashEvent      `json:"crash_event,omitempty"`
	*StageStartEvent `json:"stage_start_event,omitempty"`
	*StageEndEvent   `json:"stage_end_event,omitempty"`
}

func (e *Event) Type() string {
	switch {
	case e.StartEvent != nil:
		return e.StartEvent.Type()
	case e.MessageEvent != nil:
		return e.MessageEvent.Type()
	case e.SuccessEvent != nil:
		return e.SuccessEvent.Type()
	case e.FailureEvent != nil:
		return e.FailureEvent.Type()
	case e.CrashEvent != nil:
		return e.CrashEvent.Type()
	case e.StageStartEvent != nil:
		return e.StageStartEvent.Type()
	case e.StageEndEvent != nil:
		return e.StageEndEvent.Type()
	default:
		panic("no such event")
	}
}

type StartEvent struct {
	Runenv *RunParams `json:"runenv"`
}

func (StartEvent) Type() string {
	return "start_event"
}

func (s StartEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	return oe.AddObject("runenv", s.Runenv)
}

type MessageEvent struct {
	Message string `json:"message"`
}

func (MessageEvent) Type() string {
	return "message_event"
}

func (m MessageEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("message", m.Message)
	return nil
}

type SuccessEvent struct {
	TestGroupID string `json:"group"`
}

func (SuccessEvent) Type() string {
	return "SuccessEvent"
}

func (s SuccessEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("group", s.TestGroupID)
	return nil
}

type FailureEvent struct {
	TestGroupID string `json:"group"`
	Error       string `json:"error"`
}

func (FailureEvent) Type() string {
	return "failure_event"
}

func (f FailureEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("group", f.TestGroupID)
	oe.AddString("error", f.Error)
	return nil
}

type CrashEvent struct {
	TestGroupID string `json:"group"`
	Error       string `json:"error"`
	Stacktrace  string `json:"stacktrace"`
}

func (CrashEvent) Type() string {
	return "crash_event"
}

func (c CrashEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("group", c.TestGroupID)
	oe.AddString("error", c.Error)
	oe.AddString("stacktrace", c.Stacktrace)
	return nil
}

type StageStartEvent struct {
	Name        string `json:"name"`
	TestGroupID string `json:"group"`
}

func (StageStartEvent) Type() string {
	return "stage_start_event"
}

func (s StageStartEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("name", s.Name)
	oe.AddString("group", s.TestGroupID)
	return nil
}

type StageEndEvent struct {
	Name        string `json:"name"`
	TestGroupID string `json:"group"`
}

func (StageEndEvent) Type() string {
	return "stage_end_event"
}

func (s StageEndEvent) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("name", s.Name)
	oe.AddString("group", s.TestGroupID)
	return nil
}

func (e Event) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	switch {
	case e.StartEvent != nil:
		return oe.AddObject("start_event", e.StartEvent)
	case e.MessageEvent != nil:
		return oe.AddObject("message_event", e.MessageEvent)
	case e.SuccessEvent != nil:
		return oe.AddObject("success_event", e.SuccessEvent)
	case e.FailureEvent != nil:
		return oe.AddObject("failure_event", e.FailureEvent)
	case e.CrashEvent != nil:
		return oe.AddObject("crash_event", e.CrashEvent)
	case e.StageStartEvent != nil:
		return oe.AddObject("stage_start_event", e.StageStartEvent)
	case e.StageEndEvent != nil:
		return oe.AddObject("stage_end_event", e.StageEndEvent)
	default:
		panic("no such event")
	}
}

func (rp *RunParams) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString("plan", rp.TestPlan)
	oe.AddString("case", rp.TestCase)
	oe.AddString("run", rp.TestRun)
	if err := oe.AddReflected("params", rp.TestInstanceParams); err != nil {
		return err
	}
	oe.AddInt("instances", rp.TestInstanceCount)
	oe.AddString("outputs_path", rp.TestOutputsPath)
	oe.AddString("temp_path", rp.TestTempPath)
	oe.AddString("network", func() string {
		if rp.TestSubnet == nil {
			return ""
		}
		return rp.TestSubnet.String()
	}())

	oe.AddString("group", rp.TestGroupID)
	oe.AddInt("group_instances", rp.TestGroupInstanceCount)

	if rp.TestRepo != "" {
		oe.AddString("repo", rp.TestRepo)
	}
	if rp.TestCommit != "" {
		oe.AddString("commit", rp.TestCommit)
	}
	if rp.TestBranch != "" {
		oe.AddString("branch", rp.TestBranch)
	}
	if rp.TestTag != "" {
		oe.AddString("tag", rp.TestTag)
	}
	return nil
}

// RecordMessage records an informational message.
func (re *RunEnv) RecordMessage(msg string, a ...interface{}) {
	if len(a) > 0 {
		msg = fmt.Sprintf(msg, a...)
	}
	e := &Event{MessageEvent: &MessageEvent{
		Message: msg,
	}}
	re.logger.Info("", zap.Object("event", e))
}

func (re *RunEnv) RecordStart() {
	e := &Event{StartEvent: &StartEvent{
		Runenv: &re.RunParams,
	}}

	re.logger.Info("", zap.Object("event", e))
	re.metrics.recordEvent(e)

	_ = re.signalEmitter.SignalEvent(context.Background(), e)
}

// RecordSuccess records that the calling instance succeeded.
func (re *RunEnv) RecordSuccess() {
	e := &Event{SuccessEvent: &SuccessEvent{TestGroupID: re.RunParams.TestGroupID}}
	re.logger.Info("", zap.Object("event", e))
	re.metrics.recordEvent(e)

	_ = re.signalEmitter.SignalEvent(context.Background(), e)
}

// RecordFailure records that the calling instance failed with the supplied
// error.
func (re *RunEnv) RecordFailure(err error) {
	e := &Event{FailureEvent: &FailureEvent{TestGroupID: re.RunParams.TestGroupID, Error: err.Error()}}
	re.logger.Error("", zap.Object("event", e))
	re.metrics.recordEvent(e)

	_ = re.signalEmitter.SignalEvent(context.Background(), e)
}

// RecordCrash records that the calling instance crashed/panicked with the
// supplied error.
func (re *RunEnv) RecordCrash(err interface{}) {
	e := &Event{CrashEvent: &CrashEvent{
		TestGroupID: re.RunParams.TestGroupID,
		Error:       fmt.Sprintf("%s", err),
		Stacktrace:  string(debug.Stack()),
	}}
	re.logger.Error("", zap.Object("event", e))
	re.metrics.recordEvent(e)

	_ = re.signalEmitter.SignalEvent(context.Background(), e)
}
