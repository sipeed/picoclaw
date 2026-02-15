package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var tp *sdktrace.TracerProvider

// Init initializes OpenTelemetry tracing with an OTLP/gRPC exporter.
// The endpoint should be a Jaeger collector address (e.g. "localhost:4317").
func Init(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// Shutdown flushes and shuts down the tracer provider.
func Shutdown(ctx context.Context) error {
	if tp != nil {
		return tp.Shutdown(ctx)
	}
	return nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StringAttr is a convenience wrapper for creating a string attribute.
func StringAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// IntAttr is a convenience wrapper for creating an int attribute.
func IntAttr(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}
