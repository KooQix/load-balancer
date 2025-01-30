package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics
var (
	TotalRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_requests",
			Help: "Total number of requests received by the server",
		},
	)
	TotalRateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_rate_limit_hits",
			Help: "Number of requests blocked due to global rate limiting",
		},
	)
	PerNodeRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "per_node_requests",
			Help: "Number of requests routed to each RPC node",
		},
		[]string{"node"},
	)
	RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits",
			Help: "Number of requests blocked due to rate limiting",
		},
		[]string{"node"},
	)
	NodeErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "node_errors",
			Help: "Number of errors encountered when forwarding requests to a node",
		},
		[]string{"node"},
	)
	RequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_latency_seconds",
			Help:    "Latency of requests forwarded to RPC nodes",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"node"},
	)
)


func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(TotalRequests)
	prometheus.MustRegister(TotalRateLimitHits)
	prometheus.MustRegister(PerNodeRequests)
	prometheus.MustRegister(RateLimitHits)
	prometheus.MustRegister(NodeErrors)
	prometheus.MustRegister(RequestLatency)
}