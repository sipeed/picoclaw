package observability

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Init configures OpenTelemetry tracing. It is intentionally opt-in and
// returns a no-op shutdown function when disabled.
func Init(ctx context.Context, cfg config.ObservabilityConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	otlpExporter, err := otlptracegrpc.New(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 0.1
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			attribute.String("service.name", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(ratio)),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(otlpExporter),
	}

	if cfg.Langfuse.IsConfigured() {
		auth := base64.StdEncoding.EncodeToString([]byte(cfg.Langfuse.PublicKey + ":" + cfg.Langfuse.SecretKey))
		langfuseEndpoint := strings.TrimRight(cfg.Langfuse.Host, "/")
		switch {
		case strings.HasSuffix(langfuseEndpoint, "/api/public/otel/v1/traces"):
		case strings.HasSuffix(langfuseEndpoint, "/api/public/otel"):
			langfuseEndpoint += "/v1/traces"
		default:
			langfuseEndpoint += "/api/public/otel/v1/traces"
		}

		langfuseExporter, lfErr := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(langfuseEndpoint),
			otlptracehttp.WithHeaders(map[string]string{
				"Authorization": "Basic " + auth,
			}),
		)
		if lfErr != nil {
			logger.WarnCF("otel", "Langfuse exporter disabled (init failed)", map[string]any{
				"error": lfErr.Error(),
				"host":  cfg.Langfuse.Host,
			})
		} else {
			options = append(options, sdktrace.WithBatcher(langfuseExporter))
			logger.InfoCF("otel", "Langfuse OTLP exporter enabled", map[string]any{
				"host":     cfg.Langfuse.Host,
				"endpoint": langfuseEndpoint,
			})
		}
	}

	tp := sdktrace.NewTracerProvider(options...)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	logger.InfoCF("otel", "OpenTelemetry tracing enabled", map[string]interface{}{
		"endpoint":     cfg.OTLPEndpoint,
		"sample_ratio": ratio,
		"service_name": cfg.ServiceName,
	})

	return tp.Shutdown, nil
}

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
