package plugins

import (
	"geeksaga.com/os/straw/internal"
)

type Output interface {
	Connect() error
	Close() error
	Description() string
	SampleConfig() string
	Write(metrics []internal.Metric) error
}

type AggregatingOutput interface {
	Output
	Add(in internal.Metric)
	Push() []internal.Metric
	Reset()
}
