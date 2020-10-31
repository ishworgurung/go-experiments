package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/ishworgurung/dnsmon/mon_service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	r := flag.String("r", "", "-r [records_file]")
	t := flag.String("t", "", "-t [a|cname|srv|txt|mx|ns]")
	intv := flag.String("i", "", "-i [duration] e.g. 1h (for DNS record check interval)")
	ll := flag.String("l", "", "-l [info|warn|debug|fatal]")

	flag.Parse()
	if len(*t) == 0 || len(*r) == 0 || len(*intv) == 0 || len(*ll) == 0 {
		flag.Usage()
		return
	}

	if _, err := os.Stat(*r); err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	recordFileBytes, err := ioutil.ReadFile(*r)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	records := strings.Split(string(recordFileBytes), "\n")
	records = records[0 : len(records)-1]

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(loglevel(*ll))
	lg := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
	}()

	dnsMonitSvc, err := mon_service.New(lg, *intv)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dnsMonitSvc.Start(ctx, records, strings.ToUpper(*t)); err != nil {
			log.Error().Err(err)
		}
	}()
	wg.Wait()
}

func loglevel(l string) zerolog.Level {
	switch l {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
