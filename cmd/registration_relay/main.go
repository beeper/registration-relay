package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/libserv/pkg/flagenv"

	"github.com/beeper/registration-relay/internal/api"
	"github.com/beeper/registration-relay/internal/config"
	"github.com/beeper/registration-relay/internal/metrics"
)

var Commit,
	BuildTime string

func main() {
	prettyLogs := flag.Bool("prettyLogs", false, "Display pretty logs")
	debug := flag.Bool("debug", false, "Enable debug logging")

	listenAddr := flag.String(
		"listen",
		flagenv.StringEnvWithDefault("REGISTRATION_RELAY_LISTEN", ":8000"),
		"Listen address",
	)
	metricsListenAddr := flag.String(
		"metricsListen",
		flagenv.StringEnvWithDefault("REGISTRATION_RELAY_METRICS_LISTEN", ":5000"),
		"Metrics listen address",
	)

	validateAuthURL := flag.String(
		"validateAuthURL",
		flagenv.StringEnvWithDefault("REGISTRATION_RELAY_VALIDATE_AUTH_URL", ""),
		"Validate auth header URL",
	)

	flag.Parse()

	if *prettyLogs {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}

	cfg := config.Config{}
	cfg.API.Listen = *listenAddr
	cfg.API.ValidateAuthURL = *validateAuthURL

	log.Info().Str("commit", Commit).Str("build_time", BuildTime).Msg("registration-relay starting")

	metricsSrv := metrics.NewPrometheusMetricsHandler(*metricsListenAddr)
	metricsSrv.Start()

	srv := api.NewAPI(cfg)
	srv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c

	log.Info().Msg("Going to stop...")

	srv.Stop()
	metricsSrv.Stop()
	os.Exit(0)
}
