package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/libserv/pkg/health"
	"github.com/beeper/libserv/pkg/requestlog"

	"github.com/beeper/registration-relay/internal/config"
	"github.com/beeper/registration-relay/internal/metrics"
)

type api struct {
	log    zerolog.Logger
	server *http.Server
	secret []byte
}

func NewAPI(cfg config.Config) *api {
	logger := log.With().
		Str("component", "api").
		Logger()

	api := api{
		log:    logger,
		secret: cfg.Secret,
	}

	r := chi.NewRouter()
	r.Use(hlog.NewHandler(api.log))
	r.Use(hlog.RequestIDHandler("request_id", ""))
	r.Use(requestlog.AccessLogger(false))
	r.Use(metrics.TrackHTTPMetrics) // must be after requestlog.AccessLogger

	r.Get("/health", health.Health)

	r.Get("/api/v1/provider", api.providerWebsocket)
	r.Post("/api/v1/bridge/{command}", api.bridgeExecuteCommand)

	api.server = &http.Server{Addr: cfg.API.Listen, Handler: r}

	return &api
}

func (a *api) Start() {
	go func() {
		a.log.Info().Msgf("Starting HTTP server at: %s", a.server.Addr)

		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Fatal().Err(err).Msg("Error while listening")
		} else {
			a.log.Info().Msg("Listener stopped")
		}
	}()
}

func (a *api) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	a.log.Info().Msg("API shutdown initiated...")
	err := a.server.Shutdown(ctx)
	if err != nil {
		a.log.Fatal().Err(err).Msg("error shutting down server")
	}

	a.log.Info().Msg("API shutdown complete")
}
