package main

import (
	"context"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
	"math/rand"
	"os"
	"os/signal"
	"time"
)

func main() {
	// parse flags
	var rate float32
	flag.Float32Var(&rate, "rate", 1.0,
		"Rate to execute each scenario. e.g. 0.1 means 1 execution every 10 seconds for each scenario")

	flag.Parse()
	log.Info().Msgf("Starting canary with rate: %f ", rate)

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			cancel()
		case <-ctx.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(1)
	}()

	for action := range router.TestcasesMap {
		go run(ctx, action, rate)
	}

	<-ctx.Done()
}

func run(ctx context.Context, action string, rate float32) {
	log.Ctx(ctx).Info().Msgf("Starting scenario: %s", action)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := router.Route(ctx, models.Event{Action: action})
			if err != nil {
				log.Ctx(ctx).Error().Msg(err.Error())
			}
		}
		jitter := rand.Intn(100) - 100 // +- 100ms sleep jitter
		time.Sleep(time.Duration(1/rate)*time.Second + time.Duration(jitter)*time.Millisecond)
	}
}
