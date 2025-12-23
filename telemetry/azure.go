// Package telemetry provides observability utilities.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// AzureConfig holds Azure Application Insights configuration.
type AzureConfig struct {
	// ConnectionString is the Application Insights connection string.
	// Format: InstrumentationKey=xxx;IngestionEndpoint=https://xxx.in.applicationinsights.azure.com/
	ConnectionString string

	// ServiceName identifies the service.
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// Environment (e.g., dev, staging, prod).
	Environment string

	// SampleRate for tracing (0.0 to 1.0).
	SampleRate float64

	// MetricExportInterval is how often to export metrics.
	MetricExportInterval time.Duration
}

// DefaultAzureConfig returns default Azure configuration.
func DefaultAzureConfig() AzureConfig {
	return AzureConfig{
		SampleRate:           1.0,
		MetricExportInterval: 60 * time.Second,
	}
}

// AzureTelemetry provides Azure Application Insights integration.
type AzureTelemetry struct {
	traceProvider   *sdktrace.TracerProvider
	meterProvider   *sdkmetric.MeterProvider
	config          AzureConfig
	httpMetrics     *HTTPMetrics
	dbMetrics       map[string]*DatabaseMetrics
	businessMetrics *BusinessMetrics
}

// parseConnectionString parses an Azure Application Insights connection string.
func parseConnectionString(connStr string) (instrumentationKey, ingestionEndpoint string) {
	parts := strings.Split(connStr, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "InstrumentationKey":
			instrumentationKey = value
		case "IngestionEndpoint":
			ingestionEndpoint = strings.TrimSuffix(value, "/")
		}
	}
	return
}

// NewAzureTelemetry creates a new Azure Application Insights telemetry provider.
func NewAzureTelemetry(ctx context.Context, config AzureConfig) (*AzureTelemetry, error) {
	// Parse connection string
	instrumentationKey, ingestionEndpoint := parseConnectionString(config.ConnectionString)

	if instrumentationKey == "" {
		return nil, fmt.Errorf("missing InstrumentationKey in connection string")
	}
	if ingestionEndpoint == "" {
		// Default to global ingestion endpoint
		ingestionEndpoint = "https://dc.services.visualstudio.com"
	}

	// Create resource with service info
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
			attribute.String("ai.cloud.role", config.ServiceName),
			attribute.String("ai.cloud.roleInstance", getHostname()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace exporter
	traceEndpoint := ingestionEndpoint + "/v2/track"
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(traceEndpoint, "https://"), "http://")),
		otlptracehttp.WithHeaders(map[string]string{
			"x-ms-instrumentation-key": instrumentationKey,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create trace provider
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Create metric exporter
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(ingestionEndpoint, "https://"), "http://")),
		otlpmetrichttp.WithHeaders(map[string]string{
			"x-ms-instrumentation-key": instrumentationKey,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create metric provider
	exportInterval := config.MetricExportInterval
	if exportInterval <= 0 {
		exportInterval = 60 * time.Second
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(exportInterval))),
	)

	// Set global providers
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create common metrics
	meter := meterProvider.Meter(config.ServiceName)

	httpMetrics, err := NewHTTPMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP metrics: %w", err)
	}

	businessMetrics, err := NewBusinessMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create business metrics: %w", err)
	}

	return &AzureTelemetry{
		traceProvider:   traceProvider,
		meterProvider:   meterProvider,
		config:          config,
		httpMetrics:     httpMetrics,
		dbMetrics:       make(map[string]*DatabaseMetrics),
		businessMetrics: businessMetrics,
	}, nil
}

// Tracer returns a tracer for creating spans.
func (a *AzureTelemetry) Tracer() trace.Tracer {
	return a.traceProvider.Tracer(a.config.ServiceName)
}

// Meter returns the meter for creating instruments.
func (a *AzureTelemetry) Meter() otelmetric.Meter {
	return a.meterProvider.Meter(a.config.ServiceName)
}

// HTTPMetrics returns HTTP metrics.
func (a *AzureTelemetry) HTTPMetrics() *HTTPMetrics {
	return a.httpMetrics
}

// BusinessMetrics returns business metrics.
func (a *AzureTelemetry) BusinessMetrics() *BusinessMetrics {
	return a.businessMetrics
}

// DatabaseMetrics returns or creates database metrics for the given type.
func (a *AzureTelemetry) DatabaseMetrics(dbType string) (*DatabaseMetrics, error) {
	if m, ok := a.dbMetrics[dbType]; ok {
		return m, nil
	}

	m, err := NewDatabaseMetrics(a.meterProvider.Meter(a.config.ServiceName), dbType)
	if err != nil {
		return nil, err
	}
	a.dbMetrics[dbType] = m
	return m, nil
}

// Shutdown gracefully shuts down telemetry providers.
func (a *AzureTelemetry) Shutdown(ctx context.Context) error {
	var errs []error

	if err := a.traceProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("trace provider shutdown: %w", err))
	}

	if err := a.meterProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// getHostname returns the hostname or a default.
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// InitFromEnv initializes Azure telemetry from environment variables.
// Uses: APPLICATIONINSIGHTS_CONNECTION_STRING, SERVICE_NAME, SERVICE_VERSION, ENVIRONMENT
func InitFromEnv(ctx context.Context) (*AzureTelemetry, error) {
	connStr := os.Getenv("APPLICATIONINSIGHTS_CONNECTION_STRING")
	if connStr == "" {
		return nil, fmt.Errorf("APPLICATIONINSIGHTS_CONNECTION_STRING not set")
	}

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "cobrun-service"
	}

	serviceVersion := os.Getenv("SERVICE_VERSION")
	if serviceVersion == "" {
		serviceVersion = "1.0.0"
	}

	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	config := AzureConfig{
		ConnectionString:     connStr,
		ServiceName:          serviceName,
		ServiceVersion:       serviceVersion,
		Environment:          environment,
		SampleRate:           1.0,
		MetricExportInterval: 60 * time.Second,
	}

	return NewAzureTelemetry(ctx, config)
}

// MustInitFromEnv initializes Azure telemetry from environment or returns nil if not configured.
// This is useful for optional telemetry setup.
func MustInitFromEnv(ctx context.Context) *AzureTelemetry {
	tel, err := InitFromEnv(ctx)
	if err != nil {
		// Log but don't fail - telemetry is optional
		fmt.Printf("telemetry not initialized: %v\n", err)
		return nil
	}
	return tel
}
