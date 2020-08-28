package models

import (
	"github.com/geekflow/straw/internal"
	"github.com/geekflow/straw/plugins"
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Default size of metrics batch size.
	DEFAULT_METRIC_BATCH_SIZE = 1000

	// Default number of metrics kept. It should be a multiple of batch size.
	DEFAULT_METRIC_BUFFER_LIMIT = 10000
)

// OutputConfig containing name
type OutputConfig struct {
	Name  string
	Alias string

	FlushInterval     time.Duration
	FlushJitter       *time.Duration
	MetricBufferLimit int
	MetricBatchSize   int
}

// RunningOutput contains the output configuration
type RunningOutput struct {
	// Must be 64-bit aligned
	newMetricsCount int64
	droppedMetrics  int64

	Output            plugins.Output
	Config            *OutputConfig
	MetricBufferLimit int
	MetricBatchSize   int

	BatchReady chan time.Time

	buffer *Buffer
	//log    logger.Logger

	aggMutex sync.Mutex
}

func NewRunningOutput(
	name string,
	output plugins.Output,
	config *OutputConfig,
	batchSize int,
	bufferLimit int,
) *RunningOutput {
	tags := map[string]string{"output": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	//logger := &Logger{
	//	Name: logName("outputs", config.Name, config.Alias),
	//	Errs: selfstat.Register("write", "errors", tags),
	//}
	//setLogIfExist(output, logger)

	if config.MetricBufferLimit > 0 {
		bufferLimit = config.MetricBufferLimit
	}
	if bufferLimit == 0 {
		bufferLimit = DEFAULT_METRIC_BUFFER_LIMIT
	}
	if config.MetricBatchSize > 0 {
		batchSize = config.MetricBatchSize
	}
	if batchSize == 0 {
		batchSize = DEFAULT_METRIC_BATCH_SIZE
	}

	ro := &RunningOutput{
		buffer:            NewBuffer(config.Name, config.Alias, bufferLimit),
		BatchReady:        make(chan time.Time, 1),
		Output:            output,
		Config:            config,
		MetricBufferLimit: bufferLimit,
		MetricBatchSize:   batchSize,
		//log: logger,
	}

	return ro
}

func (r *RunningOutput) LogName() string {
	return logName("outputs", r.Config.Name, r.Config.Alias)
}

func (r *RunningOutput) Init() error {
	return nil
}

// AddMetric adds a metric to the output.
func (r *RunningOutput) AddMetric(metric internal.Metric) {

	if output, ok := r.Output.(plugins.AggregatingOutput); ok {
		r.aggMutex.Lock()
		output.Add(metric)
		r.aggMutex.Unlock()
		return
	}

	dropped := r.buffer.Add(metric)
	atomic.AddInt64(&r.droppedMetrics, int64(dropped))

	count := atomic.AddInt64(&r.newMetricsCount, 1)
	if count == int64(r.MetricBatchSize) {
		atomic.StoreInt64(&r.newMetricsCount, 0)
		select {
		case r.BatchReady <- time.Now():
		default:
		}
	}
}

// Write writes all metrics to the output, stopping when all have been sent on or error.
func (r *RunningOutput) Write() error {
	if output, ok := r.Output.(plugins.AggregatingOutput); ok {
		r.aggMutex.Lock()
		metrics := output.Push()
		r.buffer.Add(metrics...)
		output.Reset()
		r.aggMutex.Unlock()
	}

	atomic.StoreInt64(&r.newMetricsCount, 0)

	// Only process the metrics in the buffer now.  Metrics added while we are writing will be sent on the next call.
	nBuffer := r.buffer.Len()
	nBatches := nBuffer/r.MetricBatchSize + 1

	for i := 0; i < nBatches; i++ {
		batch := r.buffer.Batch(r.MetricBatchSize)
		if len(batch) == 0 {
			break
		}

		err := r.write(batch)
		if err != nil {
			r.buffer.Reject(batch)
			return err
		}
		r.buffer.Accept(batch)
	}
	return nil
}

// WriteBatch writes a single batch of metrics to the output.
func (r *RunningOutput) WriteBatch() error {
	batch := r.buffer.Batch(r.MetricBatchSize)
	if len(batch) == 0 {
		return nil
	}

	err := r.write(batch)
	if err != nil {
		r.buffer.Reject(batch)
		return err
	}
	r.buffer.Accept(batch)

	return nil
}

func (r *RunningOutput) Close() {
	err := r.Output.Close()
	if err != nil {
		log.Errorf("Error closing output: %v", err)
	}
}

func (r *RunningOutput) write(metrics []internal.Metric) error {
	dropped := atomic.LoadInt64(&r.droppedMetrics)
	if dropped > 0 {
		log.Warnf("Metric buffer overflow; %d metrics have been dropped", dropped)
		atomic.StoreInt64(&r.droppedMetrics, 0)
	}

	start := time.Now()
	err := r.Output.Write(metrics)
	elapsed := time.Since(start)
	//r.WriteTime.Incr(elapsed.Nanoseconds())

	if err == nil {
		log.Debugf("Wrote batch of %d metrics in %s", len(metrics), elapsed)
	}
	return err
}

func (r *RunningOutput) LogBufferStatus() {
	nBuffer := r.buffer.Len()
	log.Debugf("Buffer fullness: %d / %d metrics", nBuffer, r.MetricBufferLimit)
}
