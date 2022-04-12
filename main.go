package main

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	_ "github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/otel_tracer"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {

	// Set up global context with a uuid
	id, _ := uuid.NewRandom()
	ctx, cancel := context.WithCancel(context.Background())
	ctxWithId := context.WithValue(ctx, "id", id)
	defer cancel()

	// Initialize the root trace for all of Otel
	tp, cleanUpOtel := otel_tracer.InitializeOtel(ctxWithId)

	// Defer shutdown in as reasonable way as possible
	defer func() { _ = tp.Shutdown(ctxWithId) }()
	defer cleanUpOtel()

	tracer := otel.GetTracerProvider().Tracer("bacalhau.org") // if not already in scope
	otelCtx, span := tracer.Start(ctxWithId, "Main Span")

	start := time.Now()

	span.SetAttributes(attribute.String("Id", fmt.Sprintf("%s", ctxWithId.Value("id"))))
	log.Trace().Msgf("Top of execution - %s", start.UTC())
	bacalhau.Execute(VERSION, otelCtx)
	log.Trace().Msgf("Execution finished - %s", time.Since(start))
	span.End()

}
