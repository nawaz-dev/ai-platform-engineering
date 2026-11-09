package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total HTTP requests handled by the gateway.",
		},
		[]string{"method", "route", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, including streaming.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	inFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gateway_http_requests_in_flight",
			Help: "Currently active HTTP requests.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		requestsTotal,
		requestDuration,
		inFlight,
	)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

// Preserve http.Flusher so SSE streaming continues to work.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inFlight.Inc()
		defer inFlight.Dec()

		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}

		// Use a fixed route label to avoid high-cardinality metrics.
		route := "other"
		if r.URL.Path == "/v1/chat/completions" {
			route = "/v1/chat/completions"
		} else if r.URL.Path == "/v1/models" {
			route = "/v1/models"
		}

		requestsTotal.WithLabelValues(
			r.Method, route, strconv.Itoa(status),
		).Inc()

		requestDuration.WithLabelValues(
			r.Method, route,
		).Observe(time.Since(start).Seconds())
	})
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
