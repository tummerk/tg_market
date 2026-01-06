package contextx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go-backend-example/pkg/contextx"
)

func TestTraceID(t *testing.T) {
	rq := require.New(t)
	ctx := context.Background()

	var testTraceIDEmpty contextx.TraceID

	testTraceIDNotEmpty := contextx.TraceID("test-trace-id")

	traceID, err := contextx.TraceIDFromContext(ctx)
	rq.Equal(testTraceIDEmpty, traceID)
	rq.ErrorIs(err, contextx.ErrNoValue)
	rq.ErrorContains(err, "trace id: no value in context")

	ctx = contextx.WithTraceID(ctx, testTraceIDNotEmpty)

	traceID, err = contextx.TraceIDFromContext(ctx)
	rq.Equal(testTraceIDNotEmpty, traceID)
	rq.NoError(err)
}
