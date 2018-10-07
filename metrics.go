package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics stores a set of metrics recorders for monitoring
type Metrics struct {
	SearchRate  prometheus.Histogram
	ScrapeRate  prometheus.Histogram
	IndexRate   prometheus.Histogram
	ScrapeQueue prometheus.Gauge
	IndexQueue  prometheus.Gauge
	IndexSize   prometheus.Gauge
}

//nolint:lll
func newMetrics() (metrics Metrics) {
	metrics = Metrics{
		SearchRate: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "pawndex",
			Subsystem: "searcher",
			Name:      "search_rate",
			Help:      "GitHub search queries",
		}),
		ScrapeRate: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "pawndex",
			Subsystem: "scraper",
			Name:      "scrape_rate",
			Help:      "GitHub repository API accesses",
		}),
		IndexRate: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "pawndex",
			Subsystem: "indexer",
			Name:      "index_rate",
			Help:      "Pawndex index insertions",
		}),
		ScrapeQueue: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "pawndex",
			Subsystem: "scraper",
			Name:      "scrape_queue_size",
			Help:      "Size of the to-scrape queue",
		}),
		IndexQueue: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "pawndex",
			Subsystem: "indexer",
			Name:      "index_queue_size",
			Help:      "Size of the to-index queue",
		}),
		IndexSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "pawndex",
			Subsystem: "indexer",
			Name:      "index_size",
			Help:      "Overal package index size",
		}),
	}
	prometheus.MustRegister(
		metrics.SearchRate,
		metrics.ScrapeRate,
		metrics.IndexRate,
		metrics.ScrapeQueue,
		metrics.IndexQueue,
		metrics.IndexSize,
	)
	return
}
