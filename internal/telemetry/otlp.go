package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// OTLPExporter sends metrics in OpenTelemetry Protocol format
type OTLPExporter struct {
	endpoint string
	client   *http.Client
}

// NewOTLPExporter creates a new OTLP exporter
func NewOTLPExporter(endpoint string) *OTLPExporter {
	return &OTLPExporter{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// otlpMetricsPayload represents OTLP metrics in JSON format
// This is a simplified OTLP JSON representation
type otlpMetricsPayload struct {
	ResourceMetrics []otlpResourceMetrics `json:"resourceMetrics"`
}

type otlpResourceMetrics struct {
	Resource     otlpResource       `json:"resource"`
	ScopeMetrics []otlpScopeMetrics `json:"scopeMetrics"`
}

type otlpResource struct {
	Attributes []otlpAttribute `json:"attributes"`
}

type otlpScopeMetrics struct {
	Scope   otlpScope    `json:"scope"`
	Metrics []otlpMetric `json:"metrics"`
}

type otlpScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type otlpMetric struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Unit        string         `json:"unit,omitempty"`
	Sum         *otlpSum       `json:"sum,omitempty"`
	Gauge       *otlpGauge     `json:"gauge,omitempty"`
	Histogram   *otlpHistogram `json:"histogram,omitempty"`
}

type otlpSum struct {
	DataPoints             []otlpNumberDataPoint `json:"dataPoints"`
	AggregationTemporality int                   `json:"aggregationTemporality"`
	IsMonotonic            bool                  `json:"isMonotonic"`
}

type otlpGauge struct {
	DataPoints []otlpNumberDataPoint `json:"dataPoints"`
}

type otlpHistogram struct {
	DataPoints             []otlpHistogramDataPoint `json:"dataPoints"`
	AggregationTemporality int                      `json:"aggregationTemporality"`
}

type otlpNumberDataPoint struct {
	Attributes   []otlpAttribute `json:"attributes,omitempty"`
	TimeUnixNano int64           `json:"timeUnixNano"`
	AsDouble     float64         `json:"asDouble"`
}

type otlpHistogramDataPoint struct {
	Attributes     []otlpAttribute `json:"attributes,omitempty"`
	TimeUnixNano   int64           `json:"timeUnixNano"`
	Count          int64           `json:"count"`
	Sum            float64         `json:"sum"`
	BucketCounts   []int64         `json:"bucketCounts"`
	ExplicitBounds []float64       `json:"explicitBounds"`
}

type otlpAttribute struct {
	Key   string    `json:"key"`
	Value otlpValue `json:"value"`
}

type otlpValue struct {
	StringValue string `json:"stringValue,omitempty"`
}

// Export sends metrics to OTLP endpoint
func (e *OTLPExporter) Export(metrics []Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	payload := e.convertToOTLP(metrics)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal OTLP payload: %w", err)
	}

	req, err := http.NewRequest("POST", e.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("OTLP endpoint returned status %d", resp.StatusCode)
	}

	log.Debug().
		Str("endpoint", e.endpoint).
		Int("metric_count", len(metrics)).
		Int("status", resp.StatusCode).
		Msg("Successfully exported metrics via OTLP")

	return nil
}

// convertToOTLP converts our internal metrics to OTLP format
func (e *OTLPExporter) convertToOTLP(metrics []Metric) otlpMetricsPayload {
	var otlpMetrics []otlpMetric

	for _, metric := range metrics {
		timeNano := metric.Timestamp.UnixNano()

		// Convert labels to OTLP attributes
		var attributes []otlpAttribute
		for k, v := range metric.Labels {
			attributes = append(attributes, otlpAttribute{
				Key:   k,
				Value: otlpValue{StringValue: v},
			})
		}

		dataPoint := otlpNumberDataPoint{
			Attributes:   attributes,
			TimeUnixNano: timeNano,
			AsDouble:     metric.Value,
		}

		otlpMetric := otlpMetric{
			Name: metric.Name,
			Unit: metric.Unit,
		}

		switch metric.Type {
		case Counter:
			otlpMetric.Sum = &otlpSum{
				DataPoints:             []otlpNumberDataPoint{dataPoint},
				AggregationTemporality: 2, // CUMULATIVE
				IsMonotonic:            true,
			}
		case Gauge, Timer:
			otlpMetric.Gauge = &otlpGauge{
				DataPoints: []otlpNumberDataPoint{dataPoint},
			}
		case Histogram:
			// For histograms, create simple single-bucket histogram
			histDataPoint := otlpHistogramDataPoint{
				Attributes:     attributes,
				TimeUnixNano:   timeNano,
				Count:          1,
				Sum:            metric.Value,
				BucketCounts:   []int64{1},
				ExplicitBounds: []float64{},
			}
			otlpMetric.Histogram = &otlpHistogram{
				DataPoints:             []otlpHistogramDataPoint{histDataPoint},
				AggregationTemporality: 2, // CUMULATIVE
			}
		}

		otlpMetrics = append(otlpMetrics, otlpMetric)
	}

	return otlpMetricsPayload{
		ResourceMetrics: []otlpResourceMetrics{
			{
				Resource: otlpResource{
					Attributes: []otlpAttribute{
						{
							Key:   "service.name",
							Value: otlpValue{StringValue: "gaxx"},
						},
						{
							Key:   "service.version",
							Value: otlpValue{StringValue: "1.0.0"},
						},
					},
				},
				ScopeMetrics: []otlpScopeMetrics{
					{
						Scope: otlpScope{
							Name:    "gaxx-telemetry",
							Version: "1.0.0",
						},
						Metrics: otlpMetrics,
					},
				},
			},
		},
	}
}
