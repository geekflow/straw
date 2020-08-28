package models

import (
	"github.com/geekflow/straw/internal"
	"github.com/geekflow/straw/plugins"

	log "github.com/sirupsen/logrus"
	"time"
)

type RunningInput struct {
	Input  plugins.Input
	Config *InputConfig

	log         log.Logger
	defaultTags map[string]string
}

func NewRunningInput(input plugins.Input, config *InputConfig) *RunningInput {
	tags := map[string]string{"input": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	return &RunningInput{
		Input:  input,
		Config: config,
		// log:    logger,
	}
}

// InputConfig is the common config for all inputs.
type InputConfig struct {
	Name     string
	Alias    string
	Interval time.Duration

	NameOverride      string
	MeasurementPrefix string
	MeasurementSuffix string
	Tags              map[string]string
}

func (r *RunningInput) LogName() string {
	//return logName("inputs", r.Config.Name, r.Config.Alias)
	return "Log"
}

func (r *RunningInput) Init() error {
	return nil
}

func (r *RunningInput) MakeMetric(metric internal.Metric) internal.Metric {
	m := makemetric(
		metric,
		r.Config.NameOverride,
		r.Config.MeasurementPrefix,
		r.Config.MeasurementSuffix,
		r.Config.Tags,
		r.defaultTags)

	if len(metric.FieldList()) == 0 {
		return nil
	}

	return m
}

func (r *RunningInput) Gather(acc plugins.Accumulator) error {
	err := r.Input.Gather(acc)
	return err
}

func (r *RunningInput) SetDefaultTags(tags map[string]string) {
	r.defaultTags = tags
}
