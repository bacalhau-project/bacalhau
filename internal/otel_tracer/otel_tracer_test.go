package otel_tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func setupTest(t *testing.T) (context.Context, context.CancelFunc) {
	ctx := context.Background()

	// Set up global context with a uuid
	id, _ := uuid.NewRandom()
	ctx, cancel := context.WithCancel(context.Background())
	ctxWithId := context.WithValue(ctx, "id", id)

	os.Setenv("OTEL_LOCAL", "true")

	return ctxWithId, cancel
}

func teardownTest(ctxWithId context.Context, cancelFunction context.CancelFunc) {
}

func TestOtelTrace(t *testing.T) {
	ctxWithId, cancelFunction := setupTest(t)
	defer teardownTest(ctxWithId, cancelFunction)

	var w bytes.Buffer

	// Initialize the root trace for all of Otel
	_, cleanUpOtel := InitializeOtelWithWriter(ctxWithId, &w)

	tracer := otel.GetTracerProvider().Tracer("Main Trace") // if not already in scope
	otelCtx, span := tracer.Start(ctxWithId, "Main Span")

	span.SetAttributes(attribute.String("Id", fmt.Sprintf("%s", otelCtx.Value("id"))))
	span.SetAttributes(attribute.String("Start", fmt.Sprintf("%s", time.Now().UTC())))
	// bacalhau.Execute(VERSION, otelCtx)
	// log.Trace().Msgf("Execution finished - %s", time.Since(start))
	span.SetAttributes(attribute.String("End", fmt.Sprintf("%s", time.Now().UTC())))

	span.End()

	cleanUpOtel()

	var result Trace
	s := []byte(w.String())
	json.Unmarshal(s, &result)

	fmt.Print(ctxWithId)

	assert.Equal(t, result.Attributes[0].Key, "Id")
	assert.Equal(t, result.Attributes[0].Value.Value, otelCtx.Value("id").(uuid.UUID).String())
	assert.Contains(t, result.Attributes[1].Key, "Start")
	assert.Contains(t, result.Attributes[2].Key, "End")

}
