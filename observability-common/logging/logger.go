package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

func HandlerWithSpanContext(handler slog.Handler) *SpanContextLogHandler {
	return &SpanContextLogHandler{Handler: handler}
}

type SpanContextLogHandler struct {
	slog.Handler
}

func (t *SpanContextLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		record.AddAttrs(
			slog.Any("traceId", s.TraceID()),
		)
		record.AddAttrs(
			slog.Any("spanId", s.SpanID()),
		)
		record.AddAttrs(
			slog.Bool("trace_sampled", s.TraceFlags().IsSampled()),
		)
	}
	return t.Handler.Handle(ctx, record)
}
