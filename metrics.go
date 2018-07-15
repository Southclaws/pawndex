package main

import (
	"encoding/json"

	"github.com/rcrowley/go-metrics"
)

// Metrics stores a set of metrics recorders for monitoring
type Metrics struct {
	SearchRate  metrics.Meter
	ScrapeRate  metrics.Meter
	IndexRate   metrics.Meter
	ScrapeQueue metrics.Gauge
	IndexQueue  metrics.Gauge
	IndexSize   metrics.Gauge
}

func newMetrics() *Metrics {
	return &Metrics{
		SearchRate:  metrics.NewMeter(),
		ScrapeRate:  metrics.NewMeter(),
		IndexRate:   metrics.NewMeter(),
		ScrapeQueue: metrics.NewGauge(),
		IndexQueue:  metrics.NewGauge(),
		IndexSize:   metrics.NewGauge(),
	}
}

// MarshalJSON implements the JSON Unmarshaller interface
func (m Metrics) MarshalJSON() ([]byte, error) {
	object := struct {
		SearchRate  float64 `json:"searchRate"`
		ScrapeRate  float64 `json:"scrapeRate"`
		IndexRate   float64 `json:"indexRate"`
		ScrapeQueue int64   `json:"scrapeQueue"`
		IndexQueue  int64   `json:"indexQueue"`
		IndexSize   int64   `json:"indexSize"`
	}{
		SearchRate:  m.SearchRate.Rate1(),
		ScrapeRate:  m.ScrapeRate.Rate1(),
		IndexRate:   m.IndexRate.Rate1(),
		ScrapeQueue: m.ScrapeQueue.Value(),
		IndexQueue:  m.IndexQueue.Value(),
		IndexSize:   m.IndexSize.Value(),
	}
	return json.Marshal(object)
}

// UnmarshalJSON is empty because the app never needs to unmarshal metrics
func (m *Metrics) UnmarshalJSON(b []byte) error {
	return nil
}
