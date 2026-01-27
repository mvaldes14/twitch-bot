// Package telemetry contains the logging and metrics
package telemetry

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("twitch-bot")

// StartHTTPSpan creates a new span for HTTP operations with standard attributes
func StartHTTPSpan(ctx context.Context, spanName string, r *http.Request) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
			attribute.String("http.route", r.URL.Path),
			attribute.String("http.scheme", r.URL.Scheme),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
		),
	)
	return ctx, span
}

// StartSpan creates a generic span with optional attributes
func StartSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithAttributes(attrs...),
	)
	return ctx, span
}

// StartExternalSpan creates a span for external API calls
func StartExternalSpan(ctx context.Context, spanName string, service string, operation string) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("service.name", service),
			attribute.String("operation", operation),
		),
	)
	return ctx, span
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanStatus sets the status of a span based on HTTP status code
func SetSpanStatus(span trace.Span, statusCode int) {
	span.SetAttributes(attribute.Int("http.status_code", statusCode))

	if statusCode >= 400 && statusCode < 600 {
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// AddSpanAttributes adds additional attributes to a span
func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}
