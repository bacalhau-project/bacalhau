//go:build unit || !integration

package telemetry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestLoggingTracerProvider_addsSpanIDToLogger(t *testing.T) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
	})

	sb := &strings.Builder{}
	log.Logger = zerolog.New(sb)
	zerolog.DefaultContextLogger = &log.Logger

	subject := loggingTracerProvider{trace.NewTracerProvider()}
	t.Cleanup(func() {
		assert.NoError(t, subject.Shutdown(context.Background()))
	})

	ctx := context.Background()
	expectedJobId := "job-id"

	m, err := baggage.NewMember(model.TracerAttributeNameJobID, expectedJobId)
	require.NoError(t, err)
	b, err := baggage.New(m)
	require.NoError(t, err)
	ctx = baggage.ContextWithBaggage(ctx, b)

	ctx, span := subject.Tracer("test").Start(ctx, "dummy")

	log.Ctx(ctx).Error().Msg("test message")

	t.Log(sb.String())

	var message map[string]string
	require.NoError(t, json.Unmarshal([]byte(sb.String()), &message))

	assert.Equal(t, expectedJobId, message[model.TracerAttributeNameJobID])
	assert.Equal(t, span.SpanContext().SpanID().String(), message["span_id"])
	assert.Equal(t, span.SpanContext().TraceID().String(), message["trace_id"])
}
