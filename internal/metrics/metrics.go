// Package metrics holds the Prometheus collectors exposed by the radar
// server on /metrics. Today it instruments the ingest endpoint, which is
// the surface most likely to fail silently from the CronJob's side.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	IngestRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tech_radar_ingest_requests_total",
		Help: "Count of /api/v1/ingest requests, labeled by HTTP status class (2xx, 4xx, 5xx).",
	}, []string{"status"})

	IngestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tech_radar_ingest_request_duration_seconds",
		Help:    "Latency of /api/v1/ingest requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"status"})
)

// StatusClass maps an HTTP status code to a coarse label so cardinality stays bounded.
func StatusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 200:
		return "2xx"
	default:
		return "other"
	}
}

// statusRecorder captures the status code written by an http.Handler so the
// caller can record it after the handler returns. It defaults to 200 because
// http.ResponseWriter implicitly sends 200 if WriteHeader is never called.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// InstrumentIngest wraps an http.Handler with ingest counter + latency metrics.
func InstrumentIngest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		class := StatusClass(rec.status)
		IngestRequests.WithLabelValues(class).Inc()
		IngestDuration.WithLabelValues(class).Observe(time.Since(start).Seconds())
	})
}
