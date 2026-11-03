package trace

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type ShutdownFunc func(context.Context)

func InitFromEnv(ctx context.Context) ShutdownFunc {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		slog.Info("otel: no OTEL_EXPORTER_OTLP_ENDPOINT set, traces disabled")
		return func(_ context.Context) {}
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithTimeout(10*time.Second),
	)
	if err != nil {
		slog.Warn("otel: failed to create OTLP exporter, traces disabled", "error", err)
		return func(_ context.Context) {}
	}

	res := resource.NewWithAttributes(
		"https://opentelemetry.io/schemas/1.30.0",
		attribute.String("service.name", "story-builder"),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.Info("otel: initialized", "endpoint", endpoint)

	return func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("otel: tracer provider shutdown error", "error", err)
		}
	}
}
