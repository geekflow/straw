package plugins

import (
	"geeksaga.com/os/straw/internal"
	"time"
)

type Accumulator interface {
	AddFields(measurement string,
		fields map[string]interface{},
		tags map[string]string,
		t ...time.Time)

	// AddGauge is the same as AddFields, but will add the metric as a "Gauge" type
	AddGauge(measurement string,
		fields map[string]interface{},
		tags map[string]string,
		t ...time.Time)

	// AddCounter is the same as AddFields, but will add the metric as a "Counter" type
	AddCounter(measurement string,
		fields map[string]interface{},
		tags map[string]string,
		t ...time.Time)

	// SetPrecision sets the timestamp rounding precision.  All metrics addeds
	// added to the accumulator will have their timestamp rounded to the
	// nearest multiple of precision.
	SetPrecision(precision time.Duration)

	// Report an error.
	AddError(err error)
}

// TrackingID uniquely identifies a tracked metric group
type TrackingID uint64

// DeliveryInfo provides the results of a delivered metric group.
type DeliveryInfo interface {
	// ID is the TrackingID
	ID() TrackingID

	// Delivered returns true if the metric was processed successfully.
	Delivered() bool
}

// TrackingAccumulator is an Accumulator that provides a signal when the
// metric has been fully processed.  Sending more metrics than the accumulator
// has been allocated for without reading status from the Accepted or Rejected
// channels is an error.
type TrackingAccumulator interface {
	Accumulator

	// Add the Metric and arrange for tracking feedback after processing..
	AddTrackingMetric(m internal.Metric) TrackingID

	// Add a group of Metrics and arrange for a signal when the group has been
	// processed.
	AddTrackingMetricGroup(group []internal.Metric) TrackingID

	// Delivered returns a channel that will contain the tracking results.
	Delivered() <-chan DeliveryInfo
}
