package middleware

import (
	"net/http"
	"strconv"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// httpRequestDuration tracks HTTP request latency, labelled by HTTP method,
// normalized chi route pattern, and response status code.
//
// Metric name: config_gen_http_request_duration_seconds
// Labels:      method, route, status_code
//
// The `route` label uses chi.RouteContext().RoutePattern() so that dynamic
// URL segments (e.g. /{projectName}) are not expanded into high-cardinality
// label values.
var httpRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "config_gen_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds, by method, route pattern and status code.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route", "status_code"},
)

// PrometheusMetrics is chi middleware that records per-route HTTP request
// duration in the config_gen_http_request_duration_seconds histogram.
func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// RoutePattern is populated by chi after routing; read it after
		// ServeHTTP returns so the full nested pattern is available.
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unknown"
		}

		httpRequestDuration.WithLabelValues(
			r.Method,
			route,
			strconv.Itoa(ww.Status()),
		).Observe(time.Since(start).Seconds())
	})
}
