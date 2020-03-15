package internal

import (
	"time"
)

type ValueType int

const (
	_ ValueType = iota
	Counter
	Gauge
	Untyped
	Summary
	Histogram
)

type Tag struct {
	Key   string
	Value string
}

type Field struct {
	Key   string
	Value interface{}
}

type Metric interface {
	Name() string
	Tags() map[string]string
	TagList() []*Tag
	Fields() map[string]interface{}
	FieldList() []*Field
	Time() time.Time
	Type() ValueType

	// Name functions
	SetName(name string)
	AddPrefix(prefix string)
	AddSuffix(suffix string)

	// Copy returns a deep copy of the Metric.
	Copy() Metric

	// Accept marks the metric as processed successfully and written to an output.
	Accept()

	// Reject marks the metric as processed unsuccessfully.
	Reject()

	// Drop marks the metric as processed successfully without being written to any output.
	Drop()
}
