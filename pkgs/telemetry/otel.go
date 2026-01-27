// Package telemetry contains the logging and metrics
package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
)

var (
	// Global tracer provider
	tracerProvider *trace.TracerProvider
	// Global meter provider
	meterProvider *metric.MeterProvider
)

// OTELConfig holds configuration for OpenTelemetry
type OTELConfig struct {
	ServiceVersion   string
	OTLPEndpoint     string
	Environment      string
	InsecureEndpoint bool
}

// InitOTEL initializes the OpenTelemetry providers globally
func InitOTEL(ctx context.Context, config OTELConfig) error {
	// Set defaults if not provided
	if config.ServiceVersion == "" {
		config.ServiceVersion = "1.0.0"
	}
	if config.Environment == "" {
		config.Environment = "development"
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("twitch-bot"),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironmentName(config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracing and metrics if OTLP endpoint is configured
	if config.OTLPEndpoint != "" {
		if err := initTracing(ctx, res, config); err != nil {
			return fmt.Errorf("failed to initialize tracing: %w", err)
		}

		if err := initMetrics(ctx, res, config); err != nil {
			return fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	return nil
}

// initTracing sets up the global tracer provider
func initTracing(ctx context.Context, res *resource.Resource, config OTELConfig) error {
	// Create OTLP trace exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
	}

	if config.InsecureEndpoint {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider with batch span processor
	tracerProvider = trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (W3C Trace Context)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return nil
}

// initMetrics sets up the global meter provider
func initMetrics(ctx context.Context, res *resource.Resource, config OTELConfig) error {
	// Create OTLP metric exporter
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(config.OTLPEndpoint),
	}

	if config.InsecureEndpoint {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create meter provider with periodic reader
	meterProvider = metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(10*time.Second),
		)),
		metric.WithResource(res),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	return nil
}

// Shutdown gracefully shuts down the OpenTelemetry providers
func Shutdown(ctx context.Context) error {
	var err error

	if tracerProvider != nil {
		if shutdownErr := tracerProvider.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr)
		}
	}

	if meterProvider != nil {
		if shutdownErr := meterProvider.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown meter provider: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr)
			}
		}
	}

	return err
}

// GetConfigFromEnv loads OTEL configuration from environment variables
func GetConfigFromEnv() OTELConfig {
	config := OTELConfig{
		ServiceVersion:   getEnv("OTEL_SERVICE_VERSION", "1.0.0"),
		OTLPEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		Environment:      getEnv("OTEL_ENVIRONMENT", "development"),
		InsecureEndpoint: getEnv("OTEL_INSECURE_ENDPOINT", "") == "true",
	}
	return config
}

// Helper function
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
