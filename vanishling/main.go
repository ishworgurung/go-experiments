package main

import (
	"context"
	"net/http"

	"github.com/ishworgurung/vanishling/health_check"

	"github.com/ishworgurung/vanishling/cfg"

	"github.com/alecthomas/kong"
	"github.com/ishworgurung/vanishling/core"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var cli struct {
	ListenAddr string `help:"Listen address for server." default:"127.0.0.1:8080"`
	Debug      bool   `help:"Debug flag." default:false`
}

func main() {

	// add route / POST
	// o if no ttl provided use default from cfg or else use the provided ttl
	// o upload the file and store it in filesystem
	// o after the ttl expire, delete the file from fs
	// o return auth key

	// add route / GET
	// o if the auth key correct, fetch the file
	// o if the auth key incorrect, throw 4xxs

	cliCtx := kong.Parse(&cli, kong.Name("vanishling"), kong.Description("Vanishling TTL core"))

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cli.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	lg := log.Logger

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vanishling, err := core.New(ctx, cfg.DefaultLogPath, cfg.DefaultStoragePath, lg, cfg.DefaultHHSeed)
	if err != nil {
		log.Fatal().Err(err)
	}

	hc, err := health_check.New(ctx, lg)
	if err != nil {
		log.Fatal().Err(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/ping", hc)
	mux.Handle("/health", hc)
	mux.Handle("/", vanishling)

	cliCtx.FatalIfErrorf(err)
	log.Info().Msgf("Vanishling TTL core is up and running at addr `%v`", cli.ListenAddr)
	if err := http.ListenAndServe(cli.ListenAddr, mux); err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msg("Goodbye!")
}
