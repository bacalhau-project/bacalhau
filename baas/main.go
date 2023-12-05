package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/glebarez/sqlite"
	logger "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/experimental/baas/api"
	"github.com/bacalhau-project/bacalhau/experimental/baas/store"
)

func main() {
	logger.SetGlobalLevel(logger.InfoLevel)
	strg, err := store.New(store.WithDialect(sqlite.Open("baas.db")))
	if err != nil {
		panic(err)
	}

	apiV1, err := api.NewAPI(strg)
	if err != nil {
		panic(err)
	}

	svr, err := api.NewServer()
	if err != nil {
		panic(err)
	}
	svr.RegisterAPI(apiV1)

	h, err := api.NewHost(1235)
	if err != nil {
		panic(err)
	}
	fmt.Println("host ID ", h.ID().String())

	svc := api.NewService(h, apiV1)
	h.SetStreamHandler(api.ProtocolID, svc.HandleStream)

	go func() {
		if err := svr.Start(); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svr.Stop(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to stop server")
	}

}
