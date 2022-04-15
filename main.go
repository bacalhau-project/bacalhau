package main

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	_ "github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/otel_tracer"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	contextKeyName types.ContextKey = "id"
	// ...
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {

	// Set up global context with a uuid
	id, _ := uuid.NewRandom()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, contextKeyName, id) // no lint
	defer cancel()

	// Initialize the root trace for all of Otel
	tp, cleanUpOtel := otel_tracer.InitializeOtel(ctx)

	// Defer shutdown in as reasonable way as possible
	defer func() { _ = tp.Shutdown(ctx) }()
	defer cleanUpOtel()

	tracer := otel.GetTracerProvider().Tracer("bacalhau.org") // if not already in scope
	ctx, span := tracer.Start(ctx, "Main Span")

	start := time.Now()

	span.SetAttributes(attribute.String("Id", fmt.Sprintf("%s", ctx.Value("id"))))
	log.Trace().Msgf("Top of execution - %s", start.UTC())
	bacalhau.Execute(VERSION, ctx)
	log.Trace().Msgf("Execution finished - %s", time.Since(start))
	span.End()

}
