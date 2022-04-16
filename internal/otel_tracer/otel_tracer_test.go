package otel_tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
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
	os.Setenv("OTEL_STDOUT", "true") // Use the writer instead of pushing to HC, even if the key is there
	tp, cleanUpOtel := InitializeOtelWithWriter(ctxWithId, &w)

	tracer := tp.Tracer("bacalhau.org")
	otelCtx, span := tracer.Start(ctxWithId, "Main Span")

	span.SetAttributes(attribute.String("Id", fmt.Sprintf("%s", otelCtx.Value("id"))))
	span.SetAttributes(attribute.String("Start", fmt.Sprintf("%s", time.Now().UTC())))
	span.AddEvent("Test Root Event")

	_, subspan := tracer.Start(otelCtx, "Sub Span")
	subspan.SetAttributes(attribute.String("Sub Span Start", fmt.Sprintf("%s", time.Now().UTC())))
	subspan.AddEvent("Test Sub Event")
	subspan.SetAttributes(attribute.String("Sub Span End", fmt.Sprintf("%s", time.Now().UTC())))

	subspan.End()

	span.SetAttributes(attribute.String("End", fmt.Sprintf("%s", time.Now().UTC())))

	span.End()

	cleanUpOtel()

	traces := make(map[string]Trace)

	otelContent := w.String()
	if otelContent == "" {
		assert.Fail(t, "No content in the trace, did you set the environment variables correctly?")
	}
	decoder := json.NewDecoder(strings.NewReader(otelContent))
	for {
		var trace Trace

		err := decoder.Decode(&trace)
		if err == io.EOF {
			// all done
			break
		}
		if err != nil {
			log.Fatal().Msg("Failed to decode trace")
		}
		traces[trace.Name] = trace
	}

	assert.Equal(t, traces["Main Span"].Attributes[0].Key, "Id")
	assert.Equal(t, traces["Main Span"].Attributes[0].Value.Value, otelCtx.Value("id").(uuid.UUID).String())
	assert.Contains(t, traces["Main Span"].Attributes[1].Key, "Start")
	assert.Contains(t, traces["Main Span"].Attributes[2].Key, "End")
	assert.Equal(t, len(traces["Main Span"].Events), 1)

	assert.Equal(t, traces["Sub Span"].Attributes[0].Key, "Sub Span Start")
	assert.Equal(t, len(traces["Sub Span"].Events), 1)
	assert.Equal(t, traces["Sub Span"].Attributes[1].Key, "Sub Span End")

}
