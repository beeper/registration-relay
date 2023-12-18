package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/libserv/pkg/flagenv"

	"github.com/beeper/validation-relay/internal/api"
	"github.com/beeper/validation-relay/internal/config"
)

var Commit,
	BuildTime string

func main() {
	prettyLogs := flag.Bool("prettyLogs", false, "Display pretty logs")
	debug := flag.Bool("debug", false, "Enable debug logging")

	listenAddr := flag.String(
		"listen",
		flagenv.StringEnvWithDefault("BPNS_LISTEN", ":8000"),
		"Listen address",
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

	log.Info().Str("commit", Commit).Str("build_time", BuildTime).Msg("bpns")

	srv := api.NewAPI(cfg)
	srv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c

	log.Info().Msg("Going to stop...")

	srv.Stop()
	os.Exit(0)
}
