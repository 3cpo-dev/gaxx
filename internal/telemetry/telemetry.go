package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// MetricType represents the type of metric
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Timer     MetricType = "timer"
)

// Metric represents a telemetry metric
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	Unit      string            `json:"unit,omitempty"`
}

// Collector manages telemetry collection
type Collector struct {
	mu           sync.RWMutex
	metrics      []Metric
	enabled      bool
	otlpEndpoint string
	flushCh      chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewCollector creates a new telemetry collector
func NewCollector(enabled bool, otlpEndpoint string) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Collector{
		metrics:      make([]Metric, 0),
		enabled:      enabled,
		otlpEndpoint: otlpEndpoint,
		flushCh:      make(chan struct{}, 1),
		ctx:          ctx,
		cancel:       cancel,
	}

	if enabled {
		go c.periodicFlush()
	}

	return c
}

// Counter increments a counter metric
func (c *Collector) Counter(name string, value float64, labels map[string]string) {
	if !c.enabled {
		return
	}

	c.addMetric(Metric{
		Name:      name,
		Type:      Counter,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

// Gauge sets a gauge metric value
func (c *Collector) Gauge(name string, value float64, labels map[string]string) {
	if !c.enabled {
		return
	}

	c.addMetric(Metric{
		Name:      name,
		Type:      Gauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

// Histogram records a histogram value
func (c *Collector) Histogram(name string, value float64, labels map[string]string) {
	if !c.enabled {
		return
	}

	c.addMetric(Metric{
		Name:      name,
		Type:      Histogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

// Timer records a duration measurement
func (c *Collector) Timer(name string, duration time.Duration, labels map[string]string) {
	if !c.enabled {
		return
	}

	c.addMetric(Metric{
		Name:      name,
		Type:      Timer,
		Value:     float64(duration.Milliseconds()),
		Labels:    labels,
		Timestamp: time.Now(),
		Unit:      "ms",
	})
}

// addMetric adds a metric to the collection
func (c *Collector) addMetric(metric Metric) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = append(c.metrics, metric)

	// Trigger flush if we have too many metrics
	if len(c.metrics) >= 100 {
		select {
		case c.flushCh <- struct{}{}:
		default:
		}
	}
}

// GetMetrics returns a copy of current metrics
func (c *Collector) GetMetrics() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]Metric, len(c.metrics))
	copy(result, c.metrics)
	return result
}

// FlushMetrics sends metrics to the configured endpoint
func (c *Collector) FlushMetrics() error {
	c.mu.Lock()
	metrics := make([]Metric, len(c.metrics))
	copy(metrics, c.metrics)
	c.metrics = c.metrics[:0] // Clear the slice
	c.mu.Unlock()

	if len(metrics) == 0 {
		return nil
	}

	log.Debug().Int("count", len(metrics)).Msg("Flushing telemetry metrics")

	if c.otlpEndpoint != "" {
		return c.sendToOTLP(metrics)
	}

	// Fallback: log metrics
	for _, metric := range metrics {
		log.Info().
			Str("name", metric.Name).
			Str("type", string(metric.Type)).
			Float64("value", metric.Value).
			Interface("labels", metric.Labels).
			Time("timestamp", metric.Timestamp).
			Msg("telemetry_metric")
	}

	return nil
}

// sendToOTLP sends metrics to OpenTelemetry endpoint
func (c *Collector) sendToOTLP(metrics []Metric) error {
	// TODO: Implement OTLP export
	// For now, just log that we would send to OTLP
	log.Info().
		Str("endpoint", c.otlpEndpoint).
		Int("metric_count", len(metrics)).
		Msg("Would send metrics to OTLP endpoint")

	return nil
}

// periodicFlush flushes metrics every 30 seconds
func (c *Collector) periodicFlush() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			_ = c.FlushMetrics()
		case <-c.flushCh:
			_ = c.FlushMetrics()
		}
	}
}

// Shutdown stops the collector
func (c *Collector) Shutdown() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.FlushMetrics()
}

// Global collector instance
var globalCollector *Collector

// InitGlobal initializes the global telemetry collector
func InitGlobal(enabled bool, otlpEndpoint string) {
	globalCollector = NewCollector(enabled, otlpEndpoint)
}

// GetGlobal returns the global collector
func GetGlobal() *Collector {
	if globalCollector == nil {
		globalCollector = NewCollector(false, "")
	}
	return globalCollector
}

// CounterGlobal increments a counter using the global collector
func CounterGlobal(name string, value float64, labels map[string]string) {
	GetGlobal().Counter(name, value, labels)
}

// GaugeGlobal sets a gauge using the global collector
func GaugeGlobal(name string, value float64, labels map[string]string) {
	GetGlobal().Gauge(name, value, labels)
}

// HistogramGlobal records a histogram using the global collector
func HistogramGlobal(name string, value float64, labels map[string]string) {
	GetGlobal().Histogram(name, value, labels)
}

// TimerGlobal records a timer using the global collector
func TimerGlobal(name string, duration time.Duration, labels map[string]string) {
	GetGlobal().Timer(name, duration, labels)
}

// Shutdown shuts down the global collector
func Shutdown() error {
	if globalCollector != nil {
		return globalCollector.Shutdown()
	}
	return nil
}
