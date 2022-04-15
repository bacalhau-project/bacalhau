package otel_tracer

import "time"

type TraceContent struct {
	ContentsAsString string
}

type Trace struct {
	Name                   string                 `json:"Name"`
	SpanContext            SpanContext            `json:"SpanContext"`
	Parent                 Span                   `json:"Parent"`
	SpanKind               int                    `json:"SpanKind"`
	StartTime              time.Time              `json:"StartTime"`
	EndTime                time.Time              `json:"EndTime"`
	Attributes             []Attribute            `json:"Attributes"`
	Events                 []Event                `json:"Events"`
	Links                  interface{}            `json:"Links"`
	Status                 Status                 `json:"Status"`
	DroppedAttributes      int                    `json:"DroppedAttributes"`
	DroppedEvents          int                    `json:"DroppedEvents"`
	DroppedLinks           int                    `json:"DroppedLinks"`
	ChildSpanCount         int                    `json:"ChildSpanCount"`
	Resources              []Resource             `json:"Resource"`
	InstrumentationLibrary InstrumentationLibrary `json:"InstrumentationLibrary"`
}

type KeyValue struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

type Attribute struct {
	Key   string   `json:"Key"`
	Value KeyValue `json:"Value"`
}

type SpanContext struct {
	TraceID    string `json:"TraceID"`
	SpanID     string `json:"SpanID"`
	TraceFlags string `json:"TraceFlags"`
	TraceState string `json:"TraceState"`
	Remote     bool   `json:"Remote"`
}

type Span struct {
	TraceID    string `json:"TraceID"`
	SpanID     string `json:"SpanID"`
	TraceFlags string `json:"TraceFlags"`
	TraceState string `json:"TraceState"`
	Remote     bool   `json:"Remote"`
}

type Status struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}

type Resource struct {
	Key   string   `json:"Key"`
	Value KeyValue `json:"Value"`
}

type InstrumentationLibrary struct {
	Name      string `json:"Name"`
	Version   string `json:"Version"`
	SchemaURL string `json:"SchemaURL"`
}

type Event struct {
	Name                  string      `json:"Name"`
	DroppedAttributeCount int64       `json:"DroppedAttributeCount"`
	Time                  time.Time   `json:"Time"`
	Attributes            []Attribute `json:"Attributes"`
}
