package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/libserv/pkg/requestlog"
)

var (
	apiHTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "registration_relay_api_http_requests_total",
	}, []string{"path", "method", "status"})
	apiHTTPRequestDurations = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "registration_relay_api_http_request_duration_seconds",
		Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 180, 240},
	}, []string{"path", "method"})

	ProviderWebsockets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "registration_relay_provider_websockets",
		Help: "Active providers with open websockets",
	})
)

func init() {
	ProviderWebsockets.Set(0)
}

type PrometheusMetricsHandler struct {
	log    zerolog.Logger
	server *http.Server
}

func NewPrometheusMetricsHandler(listen string) *PrometheusMetricsHandler {
	logger := log.With().
		Str("component", "metrics").
		Logger()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &PrometheusMetricsHandler{
		log:    logger,
		server: &http.Server{Addr: listen, Handler: mux},
	}
}

func (mh *PrometheusMetricsHandler) Start() {
	mh.log.Info().Msgf("Starting metrics HTTP server at: %s", mh.server.Addr)
	go func() {
		err := mh.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			mh.log.Fatal().Err(err).Msg("Error in metrics listener")
		}
	}()
}

func (mh *PrometheusMetricsHandler) Stop() {
	mh.log.Info().Msg("Stopping metrics HTTP server")
	err := mh.server.Close()
	if err != nil {
		mh.log.Err(err).Msg("Error closing metrics listener")
	}
}

func TrackHTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		crw := w.(*requestlog.CountingResponseWriter)
		route := chi.RouteContext(r.Context()).RoutePattern()

		apiHTTPRequestDurations.
			With(prometheus.Labels{
				"path":   route,
				"method": r.Method,
			}).
			Observe(duration.Seconds())

		apiHTTPRequests.With(prometheus.Labels{
			"path":   route,
			"method": r.Method,
			"status": strconv.Itoa(crw.StatusCode),
		}).Inc()
	})
}
