package main

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	_ "github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/otel_tracer"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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
	ctx = context.WithValue(ctx, types.ContextId{}, id)
	defer cancel()

	// Initialize the root trace for all of Otel
	tp, cleanUpOtel := otel_tracer.GetOtelTP(ctx)

	// Defer shutdown in as reasonable way as possible
	defer func() { _ = tp.Shutdown(ctx) }()
	defer cleanUpOtel()

	tracer := tp.Tracer("bacalhau.org") // if not already in scope
	ctx, span := tracer.Start(ctx, "Main Span")

	start := time.Now()

	span.SetAttributes(attribute.String("Id", ctx.Value(types.ContextId{}).(uuid.UUID).String()))
	log.Trace().Msgf("Top of execution - %s", start.UTC())
	bacalhau.Execute(VERSION, ctx)
	log.Trace().Msgf("Execution finished - %s", time.Since(start))
	span.End()

}
