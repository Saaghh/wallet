package prometrics

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	requestsTotal           *prometheus.CounterVec
	requestDuration         *prometheus.HistogramVec
	externalRequestDuration *prometheus.HistogramVec
}

func New() *Metrics {
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"endpoint"})

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests.",
		},
		[]string{"endpoint"})

	externalRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_external_request_duration_seconds",
			Help: "Duration of external HTTP requests.",
		},
		[]string{"endpoint"})

	metrics := Metrics{
		requestsTotal:           requestsTotal,
		requestDuration:         requestDuration,
		externalRequestDuration: externalRequestDuration,
	}

	prometheus.MustRegister(
		requestsTotal,
		requestDuration,
	)

	return &metrics
}

func (m *Metrics) TrackHTTPRequest(start time.Time, r *http.Request) {
	id := chi.URLParam(r, "id")

	url := r.URL.Host + r.URL.Path
	if id != "" {
		url = strings.Replace(url, id, "{id}", 1)
	}

	method := r.Method
	elapsed := time.Since(start).Seconds()

	m.requestsTotal.WithLabelValues(method + url).Inc()
	m.requestDuration.WithLabelValues(method + url).Observe(elapsed)
}

func (m *Metrics) TrackExternalRequest(start time.Time, endpoint string) {
	elapsed := time.Since(start).Seconds()

	m.externalRequestDuration.WithLabelValues(endpoint).Observe(elapsed)
}
