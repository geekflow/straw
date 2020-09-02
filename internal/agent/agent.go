package agent

import (
	"context"
	"fmt"
	"github.com/geekflow/straw/internal"
	"github.com/geekflow/straw/internal/config"
	"github.com/geekflow/straw/internal/models"
	"github.com/geekflow/straw/plugins"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Agent runs a set of plugins.
type Agent struct {
	Config *config.Config
}

// NewAgent returns an Agent for the given Config.
func NewAgent(config *config.Config) (*Agent, error) {
	a := &Agent{
		Config: config,
	}
	return a, nil
}

// Run starts and runs the Agent until the context is done.
func (a *Agent) Run(ctx context.Context) error {
	log.Printf("[agent] Config: Interval:%s, Quiet:%#v, Hostname:%#v, "+
		"Flush Interval:%s",
		a.Config.Agent.Interval.Duration, a.Config.Agent.Quiet,
		a.Config.Agent.Hostname, a.Config.Agent.FlushInterval.Duration)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	log.Printf("[agent] Initializing plugins")
	err := a.initPlugins()
	if err != nil {
		return err
	}

	log.Printf("[agent] Connecting outputs")
	err = a.connectOutput(ctx)
	if err != nil {
		return err
	}

	inputC := make(chan internal.Metric, 100)

	startTime := time.Now()

	var wg sync.WaitGroup

	src := inputC
	dst := inputC

	wg.Add(1)
	go func(dst chan internal.Metric) {
		defer wg.Done()

		err := a.runInputs(ctx, startTime, dst)
		if err != nil {
			log.Printf("[agent] Error running inputs: %v", err)
		}

		close(dst)
		log.Printf("[agent] Input channel closed")
	}(dst)

	src = dst

	wg.Add(1)
	go func(src chan internal.Metric) {
		defer wg.Done()

		err := a.runOutputs(startTime, src)
		if err != nil {
			log.Printf("[agent] Error running outputs: %v", err)
		}
	}(src)

	wg.Wait()

	log.Printf("[agent] Closing outputs")
	a.closeOutputs()

	log.Printf("[agent] Stopped Successfully")
	return nil
}

// runInputs starts and triggers the periodic gather for Inputs.
//
// When the context is done the timers are stopped and this function returns
// after all ongoing Gather calls complete.
func (a *Agent) runInputs(
	ctx context.Context,
	startTime time.Time,
	dst chan<- internal.Metric,
) error {
	var wg sync.WaitGroup
	for _, input := range a.Config.Inputs {
		interval := a.Config.Agent.Interval.Duration
		jitter := a.Config.Agent.CollectionJitter.Duration

		// Overwrite agent interval if this plugin has its own.
		if input.Config.Interval != 0 {
			interval = input.Config.Interval
		}

		acc := NewAccumulator(input, dst)
		acc.SetPrecision(a.Precision())

		wg.Add(1)
		go func(input *models.RunningInput) {
			defer wg.Done()

			if a.Config.Agent.RoundInterval {
				err := internal.SleepContext(ctx, internal.AlignDuration(startTime, interval))
				if err != nil {
					return
				}
			}

			a.gatherOnInterval(ctx, acc, input, interval, jitter)
		}(input)
	}
	wg.Wait()

	return nil
}

// gather runs an input's gather function periodically until the context is done.
func (a *Agent) gatherOnInterval(
	ctx context.Context,
	acc plugins.Accumulator,
	input *models.RunningInput,
	interval time.Duration,
	jitter time.Duration,
) {
	defer panicRecover(input)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		err := internal.SleepContext(ctx, internal.RandomDuration(jitter))
		if err != nil {
			return
		}

		err = a.gatherOnce(acc, input, interval)
		if err != nil {
			acc.AddError(err)
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

// gatherOnce runs the input's Gather function once, logging a warning each interval it fails to complete before.
func (a *Agent) gatherOnce(
	acc plugins.Accumulator,
	input *models.RunningInput,
	timeout time.Duration,
) error {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	done := make(chan error)
	go func() {
		done <- input.Gather(acc)
	}()

	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			log.Printf("[agent] [%s] did not complete within its interval", input.LogName())
		}
	}
}

// runOutputs triggers the periodic write for Outputs.

// Runs until src is closed and all metrics have been processed.  Will call
// Write one final time before returning.
func (a *Agent) runOutputs(
	startTime time.Time,
	src <-chan internal.Metric,
) error {
	interval := a.Config.Agent.FlushInterval.Duration
	jitter := a.Config.Agent.FlushJitter.Duration

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for _, output := range a.Config.Outputs {
		interval := interval
		// Overwrite agent flush_interval if this plugin has its own.
		if output.Config.FlushInterval != 0 {
			interval = output.Config.FlushInterval
		}

		jitter := jitter
		// Overwrite agent flush_jitter if this plugin has its own.
		if output.Config.FlushJitter != nil {
			jitter = *output.Config.FlushJitter
		}

		wg.Add(1)
		go func(output *models.RunningOutput) {
			defer wg.Done()

			if a.Config.Agent.RoundInterval {
				err := internal.SleepContext(ctx, internal.AlignDuration(startTime, interval))
				if err != nil {
					return
				}
			}

			a.flush(ctx, output, interval, jitter)
		}(output)
	}

	for metric := range src {
		for i, output := range a.Config.Outputs {
			if i == len(a.Config.Outputs)-1 {
				output.AddMetric(metric)
			} else {
				output.AddMetric(metric.Copy())
			}
		}
	}

	log.Println("[agent] Hang on, flushing any cached metrics before shutdown")
	cancel()
	wg.Wait()

	return nil
}

// flush runs an output's flush function periodically until the context is done.
func (a *Agent) flush(
	ctx context.Context,
	output *models.RunningOutput,
	interval time.Duration,
	jitter time.Duration,
) {
	// since we are watching wo channels we need a ticker with the jitter integrated.
	ticker := NewTicker(interval, jitter)
	defer ticker.Stop()

	logError := func(err error) {
		if err != nil {
			log.Printf("[agent] Error writing to %s: %v", output.LogName(), err)
		}
	}

	for {
		// Favor shutdown over other methods.
		select {
		case <-ctx.Done():
			logError(a.flushOnce(output, interval, output.Write))
			return
		default:
		}

		select {
		case <-ticker.C:
			logError(a.flushOnce(output, interval, output.Write))
		case <-output.BatchReady:
			// Favor the ticker over batch ready
			select {
			case <-ticker.C:
				logError(a.flushOnce(output, interval, output.Write))
			default:
				logError(a.flushOnce(output, interval, output.WriteBatch))
			}
		case <-ctx.Done():
			logError(a.flushOnce(output, interval, output.Write))
			return
		}
	}
}

// flushOnce runs the output's Write function once, logging a warning each interval it fails to complete before.
func (a *Agent) flushOnce(
	output *models.RunningOutput,
	timeout time.Duration,
	writeFunc func() error,
) error {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	done := make(chan error)
	go func() {
		done <- writeFunc()
	}()

	for {
		select {
		case err := <-done:
			output.LogBufferStatus()
			return err
		case <-ticker.C:
			log.Printf("W! [agent] [%q] did not complete within its flush interval", output.LogName())
			output.LogBufferStatus()
		}
	}
}

// initPlugins runs the Init function on plugins.
func (a *Agent) initPlugins() error {
	for _, input := range a.Config.Inputs {
		err := input.Init()
		if err != nil {
			return fmt.Errorf("could not initialize input %s: %v",
				input.LogName(), err)
		}
	}

	for _, output := range a.Config.Outputs {
		err := output.Init()
		if err != nil {
			return fmt.Errorf("could not initialize output %s: %v",
				output.Config.Name, err)
		}
	}
	return nil
}

func (a *Agent) connectOutput(ctx context.Context) error {
	for _, output := range a.Config.Outputs {
		log.Printf("[agent] Attempting connection to [%s]", output.LogName())
		err := output.Output.Connect()
		if err != nil {
			log.Printf("[agent] Failed to connect to [%s], retrying in 15s, "+
				"error was '%s'", output.LogName(), err)

			err := internal.SleepContext(ctx, 15*time.Second)
			if err != nil {
				return err
			}

			err = output.Output.Connect()
			if err != nil {
				return err
			}
		}
		log.Printf("[agent] Successfully connected to %s", output.LogName())
	}
	return nil
}

// closeOutputs closes all outputs.
func (a *Agent) closeOutputs() {
	for _, output := range a.Config.Outputs {
		output.Close()
	}
}

// Returns the rounding precision for metrics.
func (a *Agent) Precision() time.Duration {
	precision := a.Config.Agent.Precision.Duration
	interval := a.Config.Agent.Interval.Duration

	if precision > 0 {
		return precision
	}

	switch {
	case interval >= time.Second:
		return time.Second
	case interval >= time.Millisecond:
		return time.Millisecond
	case interval >= time.Microsecond:
		return time.Microsecond
	default:
		return time.Nanosecond
	}
}

// panicRecover displays an error if an input panics.
func panicRecover(input *models.RunningInput) {
	if err := recover(); err != nil {
		trace := make([]byte, 2048)
		runtime.Stack(trace, true)
		log.Printf("FATAL: [%s] panicked: %s, Stack:\n%s", input.LogName(), err, trace)
		log.Println("PLEASE REPORT THIS PANIC ON GITHUB with " +
			"stack trace, configuration, and OS information: " +
			"https://github.com/geekflow/straw/issues/new/choose")
	}
}
