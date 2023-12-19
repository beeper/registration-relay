package main

import (
	"encoding/base64"
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
	secret := flag.String(
		"secret",
		flagenv.StringEnvWithDefault("REGISTRATION_RELAY_SECRET", ""),
		"Secret (32 bytes encoded as base64)",
	)
	metricsListenAddr := flag.String(
		"metricsListen",
		flagenv.StringEnvWithDefault("REGISTRATION_RELAY_METRICS_LISTEN", ":5000"),
		"Metrics listen address",
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
	var err error
	cfg.Secret, err = base64.StdEncoding.DecodeString(*secret)
	if err != nil || len(cfg.Secret) != 32 {
		log.Fatal().Err(err).Int("secret_len", len(cfg.Secret)).Msg("Invalid secret")
	}

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
